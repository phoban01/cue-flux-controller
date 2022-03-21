/*
Copyright 2020 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/encoding/yaml"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/runtime/acl"
	"github.com/fluxcd/pkg/runtime/events"
	"github.com/fluxcd/pkg/runtime/metrics"
	"github.com/fluxcd/pkg/runtime/predicates"
	"github.com/fluxcd/pkg/ssa"
	"github.com/fluxcd/pkg/untar"
	"github.com/hashicorp/go-retryablehttp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/cli-utils/pkg/object"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	cuev1alpha1 "github.com/phoban01/cue-flux-controller/api/v1alpha1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"

	kuberecorder "k8s.io/client-go/tools/record"
)

// CueInstanceReconciler reconciles a CueInstance object
type CueInstanceReconciler struct {
	client.Client
	httpClient            *retryablehttp.Client
	requeueDependency     time.Duration
	Scheme                *runtime.Scheme
	EventRecorder         kuberecorder.EventRecorder
	ExternalEventRecorder *events.Recorder
	MetricsRecorder       *metrics.Recorder
	StatusPoller          *polling.StatusPoller
	ControllerName        string
	statusManager         string
	NoCrossNamespaceRefs  bool
	DefaultServiceAccount string
}

// CueInstanceReconcilerOptions options
type CueInstanceReconcilerOptions struct {
	MaxConcurrentReconciles   int
	HTTPRetry                 int
	DependencyRequeueInterval time.Duration
}

//+kubebuilder:rbac:groups=cue.contrib.flux.io,resources=cueinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cue.contrib.flux.io,resources=cueinstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cue.contrib.flux.io,resources=cueinstances/finalizers,verbs=update

// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=buckets;gitrepositories,verbs=get;list;watch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=buckets/status;gitrepositories/status,verbs=get
// +kubebuilder:rbac:groups="",resources=configmaps;secrets;serviceaccounts,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// SetupWithManager sets up the controller with the Manager.
// SetupWithManager sets up the controller with the Manager.
func (r *CueInstanceReconciler) SetupWithManager(mgr ctrl.Manager, opts CueInstanceReconcilerOptions) error {
	const (
		gitRepositoryIndexKey string = ".metadata.gitRepository"
	)

	// Index the CueInstance by the GitRepository references they (may) point at.
	if err := mgr.GetCache().IndexField(context.TODO(), &cuev1alpha1.CueInstance{}, gitRepositoryIndexKey,
		r.indexBy(sourcev1.GitRepositoryKind)); err != nil {
		return fmt.Errorf("failed setting index fields: %w", err)
	}

	r.requeueDependency = opts.DependencyRequeueInterval

	r.statusManager = fmt.Sprintf("gotk-%s", r.ControllerName)

	// Configure the retryable http client used for fetching artifacts.
	// By default it retries 10 times within a 3.5 minutes window.
	httpClient := retryablehttp.NewClient()
	httpClient.RetryWaitMin = 5 * time.Second
	httpClient.RetryWaitMax = 30 * time.Second
	httpClient.RetryMax = opts.HTTPRetry
	httpClient.Logger = nil
	r.httpClient = httpClient

	return ctrl.NewControllerManagedBy(mgr).
		For(&cuev1alpha1.CueInstance{}, builder.WithPredicates(
			predicate.Or(predicate.GenerationChangedPredicate{}, predicates.ReconcileRequestedPredicate{}),
		)).
		Watches(
			&source.Kind{Type: &sourcev1.GitRepository{}},
			handler.EnqueueRequestsFromMapFunc(r.requestsForRevisionChangeOf(gitRepositoryIndexKey)),
			builder.WithPredicates(SourceRevisionChangePredicate{}),
		).
		WithOptions(controller.Options{MaxConcurrentReconciles: opts.MaxConcurrentReconciles}).
		Complete(r)
}

// Reconcile is the main reconciler loop.
func (r *CueInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	reconcileStart := time.Now()

	var cueInstance cuev1alpha1.CueInstance
	if err := r.Get(ctx, req.NamespacedName, &cueInstance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Record suspended status metric
	defer r.recordSuspension(ctx, cueInstance)

	// Add our finalizer if it does not exist
	if !controllerutil.ContainsFinalizer(&cueInstance, cuev1alpha1.CueInstanceFinalizer) {
		patch := client.MergeFrom(cueInstance.DeepCopy())
		controllerutil.AddFinalizer(&cueInstance, cuev1alpha1.CueInstanceFinalizer)
		if err := r.Patch(ctx, &cueInstance, patch); err != nil {
			log.Error(err, "unable to register finalizer")
			return ctrl.Result{}, err
		}
	}

	// Examine if the object is under deletion
	if !cueInstance.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.finalize(ctx, cueInstance)
	}

	// Return early if the CueInstance is suspended.
	if cueInstance.Spec.Suspend {
		log.Info("Reconciliation is suspended for this object")
		return ctrl.Result{}, nil
	}

	// resolve source reference
	source, err := r.getSource(ctx, cueInstance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			msg := fmt.Sprintf("Source '%s' not found", cueInstance.Spec.SourceRef.String())
			cueInstance = cuev1alpha1.CueInstanceNotReady(cueInstance, "", cuev1alpha1.ArtifactFailedReason, msg)
			if err := r.patchStatus(ctx, req, cueInstance.Status); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			r.recordReadiness(ctx, cueInstance)
			log.Info(msg)
			// do not requeue immediately, when the source is created the watcher should trigger a reconciliation
			return ctrl.Result{RequeueAfter: cueInstance.GetRetryInterval()}, nil
		}

		// retry on transient errors
		return ctrl.Result{Requeue: true}, err

	}

	if source.GetArtifact() == nil {
		msg := "Source is not ready, artifact not found"
		cueInstance = cuev1alpha1.CueInstanceNotReady(cueInstance, "", cuev1alpha1.ArtifactFailedReason, msg)
		if err := r.patchStatus(ctx, req, cueInstance.Status); err != nil {
			log.Error(err, "unable to update status for artifact not found")
			return ctrl.Result{Requeue: true}, err
		}
		r.recordReadiness(ctx, cueInstance)
		log.Info(msg)
		// do not requeue immediately, when the artifact is created the watcher should trigger a reconciliation
		return ctrl.Result{RequeueAfter: cueInstance.GetRetryInterval()}, nil
	}

	// check dependencies
	if len(cueInstance.Spec.DependsOn) > 0 {
		if err := r.checkDependencies(source, cueInstance); err != nil {
			cueInstance = cuev1alpha1.CueInstanceNotReady(
				cueInstance, source.GetArtifact().Revision, meta.FailedReason, err.Error())
			if err := r.patchStatus(ctx, req, cueInstance.Status); err != nil {
				log.Error(err, "unable to update status for dependency not ready")
				return ctrl.Result{Requeue: true}, err
			}
			// we can't rely on exponential backoff because it will prolong the execution too much,
			// instead we requeue on a fix interval.
			msg := fmt.Sprintf("Dependencies do not meet ready condition, retrying in %s", r.requeueDependency.String())
			log.Info(msg)
			r.event(ctx, cueInstance, source.GetArtifact().Revision, events.EventSeverityInfo, msg, nil)
			r.recordReadiness(ctx, cueInstance)
			return ctrl.Result{RequeueAfter: r.requeueDependency}, nil
		}
		log.Info("All dependencies are ready, proceeding with reconciliation")
	}

	// record reconciliation duration
	if r.MetricsRecorder != nil {
		objRef, err := reference.GetReference(r.Scheme, &cueInstance)
		if err != nil {
			return ctrl.Result{}, err
		}
		defer r.MetricsRecorder.RecordDuration(*objRef, reconcileStart)
	}

	// set the reconciliation status to progressing
	cueInstance = cuev1alpha1.CueInstanceProgressing(cueInstance, "reconciliation in progress")
	if err := r.patchStatus(ctx, req, cueInstance.Status); err != nil {
		return ctrl.Result{Requeue: true}, err
	}
	r.recordReadiness(ctx, cueInstance)

	// reconcile cueInstance by applying the latest revision
	reconciledCueInstance, reconcileErr := r.reconcile(ctx, *cueInstance.DeepCopy(), source)
	if err := r.patchStatus(ctx, req, reconciledCueInstance.Status); err != nil {
		return ctrl.Result{Requeue: true}, err
	}
	r.recordReadiness(ctx, reconciledCueInstance)

	// broadcast the reconciliation failure and requeue at the specified retry interval
	if reconcileErr != nil {
		log.Error(reconcileErr, fmt.Sprintf("Reconciliation failed after %s, next try in %s",
			time.Since(reconcileStart).String(),
			cueInstance.GetRetryInterval().String()),
			"revision",
			source.GetArtifact().Revision)
		r.event(ctx, reconciledCueInstance, source.GetArtifact().Revision, events.EventSeverityError,
			reconcileErr.Error(), nil)
		return ctrl.Result{RequeueAfter: cueInstance.GetRetryInterval()}, nil
	}

	// broadcast the reconciliation result and requeue at the specified interval
	msg := fmt.Sprintf("Reconciliation finished in %s, next run in %s",
		time.Since(reconcileStart).String(),
		cueInstance.Spec.Interval.Duration.String())
	log.Info(msg, "revision", source.GetArtifact().Revision)
	r.event(ctx, reconciledCueInstance, source.GetArtifact().Revision, events.EventSeverityInfo,
		msg, map[string]string{"commit_status": "update"})

	return ctrl.Result{RequeueAfter: cueInstance.Spec.Interval.Duration}, nil
}

func (r *CueInstanceReconciler) reconcile(
	ctx context.Context,
	cueInstance cuev1alpha1.CueInstance,
	source sourcev1.Source) (cuev1alpha1.CueInstance, error) {
	// record the value of the reconciliation request, if any
	if v, ok := meta.ReconcileAnnotationValue(cueInstance.GetAnnotations()); ok {
		cueInstance.Status.SetLastHandledReconcileRequest(v)
	}

	revision := source.GetArtifact().Revision

	// create tmp dir
	tmpDir, err := os.MkdirTemp("", cueInstance.Name)
	if err != nil {
		err = fmt.Errorf("tmp dir error: %w", err)
		return cuev1alpha1.CueInstanceNotReady(
			cueInstance,
			revision,
			sourcev1.DirCreationFailedReason,
			err.Error(),
		), err
	}
	defer os.RemoveAll(tmpDir)

	// download artifact and extract files
	err = r.download(source.GetArtifact(), tmpDir)
	if err != nil {
		return cuev1alpha1.CueInstanceNotReady(
			cueInstance,
			revision,
			cuev1alpha1.ArtifactFailedReason,
			err.Error(),
		), err
	}

	// check module path exists
	moduleRootPath, err := securejoin.SecureJoin(tmpDir, cueInstance.Spec.Root)
	if err != nil {
		return cuev1alpha1.CueInstanceNotReady(
			cueInstance,
			revision,
			cuev1alpha1.ArtifactFailedReason,
			err.Error(),
		), err
	}

	if _, err := os.Stat(moduleRootPath); err != nil {
		err = fmt.Errorf("cueInstance module root path not found: %w", err)
		return cuev1alpha1.CueInstanceNotReady(
			cueInstance,
			revision,
			cuev1alpha1.ArtifactFailedReason,
			err.Error(),
		), err
	}
	// check build path exists
	dirPath, err := securejoin.SecureJoin(moduleRootPath, cueInstance.Spec.Path)
	if err != nil {
		return cuev1alpha1.CueInstanceNotReady(
			cueInstance,
			revision,
			cuev1alpha1.ArtifactFailedReason,
			err.Error(),
		), err
	}

	if _, err := os.Stat(dirPath); err != nil {
		err = fmt.Errorf("cueInstance path not found: %w", err)
		return cuev1alpha1.CueInstanceNotReady(
			cueInstance,
			revision,
			cuev1alpha1.ArtifactFailedReason,
			err.Error(),
		), err
	}

	// setup a Kubernetes client
	// setup the Kubernetes client for impersonation
	impersonation := NewCueInstanceImpersonation(cueInstance, r.Client, r.StatusPoller, r.DefaultServiceAccount)
	kubeClient, statusPoller, err := impersonation.GetClient(ctx)
	if err != nil {
		return cuev1alpha1.CueInstanceNotReady(
			cueInstance,
			revision,
			meta.FailedReason,
			err.Error(),
		), fmt.Errorf("failed to build kube client: %w", err)
	}

	// build the cueInstance
	resources, err := r.build(ctx, revision, moduleRootPath, dirPath, &cueInstance)
	if err != nil {
		return cuev1alpha1.CueInstanceNotReady(
			cueInstance,
			revision,
			cuev1alpha1.BuildFailedReason,
			err.Error(),
		), err
	}

	// convert the build result into Kubernetes unstructured objects
	objects, err := ssa.ReadObjects(bytes.NewReader(resources))
	if err != nil {
		return cuev1alpha1.CueInstanceNotReady(
			cueInstance,
			revision,
			cuev1alpha1.BuildFailedReason,
			err.Error(),
		), err
	}

	// create a snapshot of the current inventory
	oldStatus := cueInstance.Status.DeepCopy()

	// create the server-side apply manager
	resourceManager := ssa.NewResourceManager(kubeClient, statusPoller, ssa.Owner{
		Field: r.ControllerName,
		Group: cuev1alpha1.GroupVersion.Group,
	})
	resourceManager.SetOwnerLabels(objects, cueInstance.GetName(), cueInstance.GetNamespace())

	// validate and apply resources in stages
	drifted, changeSet, err := r.apply(ctx, resourceManager, cueInstance, revision, objects)
	if err != nil {
		return cuev1alpha1.CueInstanceNotReady(
			cueInstance,
			revision,
			meta.FailedReason,
			err.Error(),
		), err
	}

	// create an inventory of objects to be reconciled
	newInventory := NewInventory()
	err = AddObjectsToInventory(newInventory, changeSet)
	if err != nil {
		return cuev1alpha1.CueInstanceNotReady(
			cueInstance,
			revision,
			meta.FailedReason,
			err.Error(),
		), err
	}

	// detect stale objects which are subject to garbage collection
	var staleObjects []*unstructured.Unstructured
	if oldStatus.Inventory != nil {
		diffObjects, err := DiffInventory(oldStatus.Inventory, newInventory)
		if err != nil {
			return cuev1alpha1.CueInstanceNotReady(
				cueInstance,
				revision,
				meta.FailedReason,
				err.Error(),
			), err
		}

		// TODO: remove this workaround after kustomize-controller 0.18 release
		// skip objects that were wrongly marked as namespaced
		// https://github.com/fluxcd/kustomize-controller/issues/466
		newObjects, _ := ListObjectsInInventory(newInventory)
		for _, obj := range diffObjects {
			preserve := false
			if obj.GetNamespace() != "" {
				for _, newObj := range newObjects {
					if newObj.GetNamespace() == "" &&
						obj.GetKind() == newObj.GetKind() &&
						obj.GetAPIVersion() == newObj.GetAPIVersion() &&
						obj.GetName() == newObj.GetName() {
						preserve = true
						break
					}
				}
			}
			if !preserve {
				staleObjects = append(staleObjects, obj)
			}
		}
	}

	// run garbage collection for stale objects that do not have pruning disabled
	if _, err := r.prune(ctx, resourceManager, cueInstance, revision, staleObjects); err != nil {
		return cuev1alpha1.CueInstanceNotReadyInventory(
			cueInstance,
			newInventory,
			revision,
			cuev1alpha1.PruneFailedReason,
			err.Error(),
		), err
	}

	// health assessment
	if err := r.checkHealth(ctx, resourceManager, cueInstance, revision, drifted, changeSet.ToObjMetadataSet()); err != nil {
		return cuev1alpha1.CueInstanceNotReadyInventory(
			cueInstance,
			newInventory,
			revision,
			cuev1alpha1.HealthCheckFailedReason,
			err.Error(),
		), err
	}

	return cuev1alpha1.CueInstanceReadyInventory(
		cueInstance,
		newInventory,
		revision,
		meta.FailedReason,
		fmt.Sprintf("Applied revision: %s", revision),
	), err
}

func (r *CueInstanceReconciler) build(ctx context.Context,
	revision, root, dir string,
	instance *cuev1alpha1.CueInstance,
) ([]byte, error) {
	log := ctrl.LoggerFrom(ctx)
	cctx := cuecontext.New()

	tags := make([]string, 0, len(instance.Spec.Tags))
	for _, t := range instance.Spec.Tags {
		if t.Value != "" {
			tags = append(tags, fmt.Sprintf("%s=%s", t.Name, t.Value))
		} else {
			tags = append(tags, t.Name)
		}
	}

	tagVars := load.DefaultTagVars()
	for _, t := range instance.Spec.TagVars {
		tagVars[t.Name] = load.TagVar{
			Func: func() (ast.Expr, error) {
				return ast.NewString(t.Value), nil
			},
		}
	}

	cfg := &load.Config{
		ModuleRoot: root,
		Dir:        dir,
		DataFiles:  true, //TODO: this could be configurable
		Tags:       tags,
		TagVars:    tagVars,
	}

	if instance.Spec.Package != "" {
		cfg.Package = instance.Spec.Package
	}

	ix := load.Instances([]string{}, cfg)
	if len(ix) == 0 {
		return nil, fmt.Errorf("no instances found")
	}

	inst := ix[0]
	if inst.Err != nil {
		return nil, inst.Err
	}

	value := cctx.BuildInstance(inst)
	if value.Err() != nil {
		return nil, value.Err()
	}

	shouldValidate := instance.Spec.Validate != nil

	var result bytes.Buffer
	if len(instance.Spec.Exprs) > 0 {
		for _, e := range instance.Spec.Exprs {
			expr := value.LookupPath(cue.ParsePath(e))

			data, err := cueEncodeYAML(expr)
			if err != nil {
				return nil, err
			}

			if shouldValidate && instance.Spec.Validate.Type == "cue" {
				schema := value.LookupPath(cue.ParsePath(instance.Spec.Validate.Schema))
				schema.Unify(expr)
				if err := schema.Unify(expr).Validate(); err != nil {
					msg := fmt.Sprintf("cue expression validation failed: %s", err)
					switch instance.Spec.Validate.Mode {
					case cuev1alpha1.FailPolicy:
						r.event(ctx, *instance, revision, events.EventSeverityInfo, msg, nil)
						return nil, fmt.Errorf(msg)
					case cuev1alpha1.DropPolicy:
						r.event(ctx, *instance, revision, events.EventSeverityInfo, msg, nil)
						continue
					case cuev1alpha1.AuditPolicy:
						r.event(ctx, *instance, revision, events.EventSeverityInfo, msg, nil)
						break
					case cuev1alpha1.IgnorePolicy:
						log.Info(msg)
						break
					}
				}
			}

			_, err = result.Write(data)
			if err != nil {
				return nil, err
			}
		}
	} else {
		data, err := cueEncodeYAML(value)
		if err != nil {
			return nil, err
		}

		valid := false

		if shouldValidate && instance.Spec.Validate.Type == "cue" {
			schema := value.LookupPath(cue.ParsePath(instance.Spec.Validate.Schema))
			if err := schema.Unify(value).Validate(); err != nil {
				msg := fmt.Sprintf("cue validation failed: %s", err)
				switch instance.Spec.Validate.Mode {
				case cuev1alpha1.FailPolicy:
					r.event(ctx, *instance, revision, events.EventSeverityInfo, msg, nil)
					return nil, fmt.Errorf(msg)
				case cuev1alpha1.DropPolicy:
					r.event(ctx, *instance, revision, events.EventSeverityInfo, msg, nil)
					valid = false
				case cuev1alpha1.AuditPolicy:
					r.event(ctx, *instance, revision, events.EventSeverityInfo, msg, nil)
				case cuev1alpha1.IgnorePolicy:
					log.Info(msg)
				}
			}
		}

		if valid {
			_, err = result.Write(data)
			if err != nil {
				return nil, err
			}
		}
	}

	for _, of := range inst.OrphanedFiles {
		if of.Encoding == "yaml" {
			data, err := yaml.Extract(of.Filename, nil)
			if err != nil {
				return nil, err
			}
			f := cctx.BuildFile(data)
			switch f.Kind() {
			case cue.ListKind:
				l, err := f.List()
				if err != nil {
					return nil, err
				}
				for l.Next() {
					data, err := yaml.Encode(l.Value())
					if err != nil {
						return nil, err
					}
					if shouldValidate && instance.Spec.Validate.Type == "yaml" {
						schema := value.LookupPath(cue.ParsePath(instance.Spec.Validate.Schema))
						if err := yaml.Validate(data, schema); err != nil {
							msg := fmt.Sprintf("yaml validation failed: %s", err)
							switch instance.Spec.Validate.Mode {
							case cuev1alpha1.FailPolicy:
								r.event(ctx, *instance, revision, events.EventSeverityInfo, msg, nil)
								return nil, fmt.Errorf(msg)
							case cuev1alpha1.DropPolicy:
								r.event(ctx, *instance, revision, events.EventSeverityInfo, msg, nil)
								continue
							case cuev1alpha1.AuditPolicy:
								r.event(ctx, *instance, revision, events.EventSeverityInfo, msg, nil)
								break
							case cuev1alpha1.IgnorePolicy:
								log.Info(msg)
							}
						}
					}
					result.Write(data)
					result.Write([]byte("\n---\n"))
				}
			case cue.StructKind:
				data, err := yaml.Encode(f)
				if err != nil {
					return nil, err
				}
				if shouldValidate && instance.Spec.Validate.Type == "yaml" {
					schema := value.LookupPath(cue.ParsePath(instance.Spec.Validate.Schema))
					if err := yaml.Validate(data, schema); err != nil {
						msg := fmt.Sprintf("yaml validation failed: %s", err)
						switch instance.Spec.Validate.Mode {
						case cuev1alpha1.FailPolicy:
							r.event(ctx, *instance, revision, events.EventSeverityInfo, msg, nil)
							return nil, fmt.Errorf(msg)
						case cuev1alpha1.DropPolicy:
							r.event(ctx, *instance, revision, events.EventSeverityInfo, msg, nil)
							continue
						case cuev1alpha1.AuditPolicy:
							r.event(ctx, *instance, revision, events.EventSeverityInfo, msg, nil)
						case cuev1alpha1.IgnorePolicy:
							log.Info(msg)
						}
					}
				}
				result.Write(data)
				result.Write([]byte("\n---\n"))
			}
		}
	}

	return result.Bytes(), nil
}

func cueEncodeYAML(value cue.Value) ([]byte, error) {
	var (
		err  error
		data []byte
	)
	switch value.Kind() {
	case cue.ListKind:
		items, err := value.List()
		if err != nil {
			return nil, err
		}
		data, err = yaml.EncodeStream(items)
		if err != nil {
			return nil, err
		}
	case cue.StructKind:
		data, err = yaml.Encode(value)
		if err != nil {
			return nil, err
		}
	default:
		return nil, nil
	}
	data = append(data, []byte("\n---\n")...)
	return data, nil
}

func (r *CueInstanceReconciler) apply(ctx context.Context, manager *ssa.ResourceManager, cueInstance cuev1alpha1.CueInstance, revision string, objects []*unstructured.Unstructured) (bool, *ssa.ChangeSet, error) {
	log := ctrl.LoggerFrom(ctx)

	if err := ssa.SetNativeKindsDefaults(objects); err != nil {
		return false, nil, err
	}

	applyOpts := ssa.DefaultApplyOptions()
	applyOpts.Exclusions = map[string]string{
		fmt.Sprintf("%s/reconcile", cuev1alpha1.GroupVersion.Group): cuev1alpha1.DisabledValue,
	}

	// contains only CRDs and Namespaces
	var stageOne []*unstructured.Unstructured

	// contains all objects except for CRDs and Namespaces
	var stageTwo []*unstructured.Unstructured

	// contains the objects' metadata after apply
	resultSet := ssa.NewChangeSet()

	for _, u := range objects {
		if ssa.IsClusterDefinition(u) {
			stageOne = append(stageOne, u)
		} else {
			stageTwo = append(stageTwo, u)
		}
	}

	var changeSetLog strings.Builder

	// validate, apply and wait for CRDs and Namespaces to register
	if len(stageOne) > 0 {
		changeSet, err := manager.ApplyAll(ctx, stageOne, applyOpts)
		if err != nil {
			return false, nil, err
		}
		resultSet.Append(changeSet.Entries)

		if changeSet != nil && len(changeSet.Entries) > 0 {
			log.Info("server-side apply completed", "output", changeSet.ToMap())
			for _, change := range changeSet.Entries {
				if change.Action != string(ssa.UnchangedAction) {
					changeSetLog.WriteString(change.String() + "\n")
				}
			}
		}

		if err := manager.Wait(stageOne, ssa.WaitOptions{
			Interval: 2 * time.Second,
			Timeout:  cueInstance.GetTimeout(),
		}); err != nil {
			return false, nil, err
		}
	}

	// sort by kind, validate and apply all the others objects
	sort.Sort(ssa.SortableUnstructureds(stageTwo))
	if len(stageTwo) > 0 {
		changeSet, err := manager.ApplyAll(ctx, stageTwo, applyOpts)
		if err != nil {
			return false, nil, fmt.Errorf("%w\n%s", err, changeSetLog.String())
		}
		resultSet.Append(changeSet.Entries)

		if changeSet != nil && len(changeSet.Entries) > 0 {
			log.Info("server-side apply completed", "output", changeSet.ToMap())
			for _, change := range changeSet.Entries {
				if change.Action != string(ssa.UnchangedAction) {
					changeSetLog.WriteString(change.String() + "\n")
				}
			}
		}
	}

	// emit event only if the server-side apply resulted in changes
	applyLog := strings.TrimSuffix(changeSetLog.String(), "\n")
	if applyLog != "" {
		r.event(ctx, cueInstance, revision, events.EventSeverityInfo, applyLog, nil)
	}

	return applyLog != "", resultSet, nil
}

func (r *CueInstanceReconciler) checkDependencies(source sourcev1.Source, cueInstance cuev1alpha1.CueInstance) error {
	for _, d := range cueInstance.Spec.DependsOn {
		if d.Namespace == "" {
			d.Namespace = cueInstance.GetNamespace()
		}
		dName := types.NamespacedName{
			Namespace: d.Namespace,
			Name:      d.Name,
		}
		var k cuev1alpha1.CueInstance
		err := r.Get(context.Background(), dName, &k)
		if err != nil {
			return fmt.Errorf("unable to get '%s' dependency: %w", dName, err)
		}

		if len(k.Status.Conditions) == 0 || k.Generation != k.Status.ObservedGeneration {
			return fmt.Errorf("dependency '%s' is not ready", dName)
		}

		if !apimeta.IsStatusConditionTrue(k.Status.Conditions, meta.ReadyCondition) {
			return fmt.Errorf("dependency '%s' is not ready", dName)
		}

		if k.Spec.SourceRef.Name == cueInstance.Spec.SourceRef.Name && k.Spec.SourceRef.Namespace == cueInstance.Spec.SourceRef.Namespace && k.Spec.SourceRef.Kind == cueInstance.Spec.SourceRef.Kind && source.GetArtifact().Revision != k.Status.LastAppliedRevision {
			return fmt.Errorf("dependency '%s' is not updated yet", dName)
		}
	}

	return nil
}

func (r *CueInstanceReconciler) download(artifact *sourcev1.Artifact, tmpDir string) error {
	artifactURL := artifact.URL
	if hostname := os.Getenv("SOURCE_CONTROLLER_LOCALHOST"); hostname != "" {
		u, err := url.Parse(artifactURL)
		if err != nil {
			return err
		}
		u.Host = hostname
		artifactURL = u.String()
	}

	req, err := retryablehttp.NewRequest(http.MethodGet, artifactURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create a new request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download artifact, error: %w", err)
	}
	defer resp.Body.Close()

	// check response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download artifact from %s, status: %s", artifactURL, resp.Status)
	}

	var buf bytes.Buffer

	// verify checksum matches origin
	if err := r.verifyArtifact(artifact, &buf, resp.Body); err != nil {
		return err
	}

	// extract
	if _, err = untar.Untar(&buf, tmpDir); err != nil {
		return fmt.Errorf("failed to untar artifact, error: %w", err)
	}

	return nil
}

func (r *CueInstanceReconciler) verifyArtifact(artifact *sourcev1.Artifact, buf *bytes.Buffer, reader io.Reader) error {
	hasher := sha256.New()

	// for backwards compatibility with source-controller v0.17.2 and older
	if len(artifact.Checksum) == 40 {
		hasher = sha1.New()
	}

	// compute checksum
	mw := io.MultiWriter(hasher, buf)
	if _, err := io.Copy(mw, reader); err != nil {
		return err
	}

	if checksum := fmt.Sprintf("%x", hasher.Sum(nil)); checksum != artifact.Checksum {
		return fmt.Errorf("failed to verify artifact: computed checksum '%s' doesn't match advertised '%s'",
			checksum, artifact.Checksum)
	}

	return nil
}

func (r *CueInstanceReconciler) prune(ctx context.Context, manager *ssa.ResourceManager, cueInstance cuev1alpha1.CueInstance, revision string, objects []*unstructured.Unstructured) (bool, error) {
	if !cueInstance.Spec.Prune {
		return false, nil
	}

	log := ctrl.LoggerFrom(ctx)

	opts := ssa.DeleteOptions{
		PropagationPolicy: metav1.DeletePropagationBackground,
		Inclusions:        manager.GetOwnerLabels(cueInstance.Name, cueInstance.Namespace),
		Exclusions: map[string]string{
			fmt.Sprintf("%s/prune", cuev1alpha1.GroupVersion.Group):     cuev1alpha1.DisabledValue,
			fmt.Sprintf("%s/reconcile", cuev1alpha1.GroupVersion.Group): cuev1alpha1.DisabledValue,
		},
	}

	changeSet, err := manager.DeleteAll(ctx, objects, opts)
	if err != nil {
		return false, err
	}

	// emit event only if the prune operation resulted in changes
	if changeSet != nil && len(changeSet.Entries) > 0 {
		log.Info(fmt.Sprintf("garbage collection completed: %s", changeSet.String()))
		r.event(ctx, cueInstance, revision, events.EventSeverityInfo, changeSet.String(), nil)
		return true, nil
	}

	return false, nil
}

func (r *CueInstanceReconciler) checkHealth(ctx context.Context, manager *ssa.ResourceManager, cueInstance cuev1alpha1.CueInstance, revision string, drifted bool, objects object.ObjMetadataSet) error {
	if len(cueInstance.Spec.HealthChecks) == 0 && !cueInstance.Spec.Wait {
		return nil
	}

	checkStart := time.Now()
	var err error
	if !cueInstance.Spec.Wait {
		objects, err = referenceToObjMetadataSet(cueInstance.Spec.HealthChecks)
		if err != nil {
			return err
		}
	}

	if len(objects) == 0 {
		return nil
	}

	// guard against deadlock (waiting on itself)
	var toCheck []object.ObjMetadata
	for _, object := range objects {
		if object.GroupKind.Kind == cuev1alpha1.CueInstanceKind &&
			object.Name == cueInstance.GetName() &&
			object.Namespace == cueInstance.GetNamespace() {
			continue
		}
		toCheck = append(toCheck, object)
	}

	// find the previous health check result
	wasHealthy := apimeta.IsStatusConditionTrue(cueInstance.Status.Conditions, cuev1alpha1.HealthyCondition)

	// set the Healthy and Ready conditions to progressing
	message := fmt.Sprintf("running health checks with a timeout of %s", cueInstance.GetTimeout().String())
	c := cuev1alpha1.CueInstanceProgressing(cueInstance, message)
	cuev1alpha1.SetCueInstanceHealthiness(&c, metav1.ConditionUnknown, meta.ProgressingReason, message)
	if err := r.patchStatus(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&cueInstance)}, c.Status); err != nil {
		return fmt.Errorf("unable to update the healthy status to progressing, error: %w", err)
	}

	// check the health with a default timeout of 30sec shorter than the reconciliation interval
	if err := manager.WaitForSet(toCheck, ssa.WaitOptions{
		Interval: 5 * time.Second,
		Timeout:  cueInstance.GetTimeout(),
	}); err != nil {
		return fmt.Errorf("Health check failed after %s, %w", time.Since(checkStart).String(), err)
	}

	// emit event if the previous health check failed
	if !wasHealthy || (cueInstance.Status.LastAppliedRevision != revision && drifted) {
		r.event(ctx, cueInstance, revision, events.EventSeverityInfo,
			fmt.Sprintf("Health check passed in %s", time.Since(checkStart).String()), nil)
	}

	return nil
}

func (r *CueInstanceReconciler) getSource(ctx context.Context, cueInstance cuev1alpha1.CueInstance) (sourcev1.Source, error) {
	var source sourcev1.Source
	sourceNamespace := cueInstance.GetNamespace()
	if cueInstance.Spec.SourceRef.Namespace != "" {
		sourceNamespace = cueInstance.Spec.SourceRef.Namespace
	}

	namespacedName := types.NamespacedName{
		Namespace: sourceNamespace,
		Name:      cueInstance.Spec.SourceRef.Name,
	}

	if r.NoCrossNamespaceRefs && sourceNamespace != cueInstance.GetNamespace() {
		return source, acl.AccessDeniedError(
			fmt.Sprintf("can't access '%s/%s', cross-namespace references have been blocked",
				cueInstance.Spec.SourceRef.Kind, namespacedName))
	}

	switch cueInstance.Spec.SourceRef.Kind {
	case sourcev1.GitRepositoryKind:
		var repository sourcev1.GitRepository
		err := r.Client.Get(ctx, namespacedName, &repository)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return source, err
			}
			return source, fmt.Errorf("unable to get source '%s': %w", namespacedName, err)
		}
		source = &repository
	default:
		return source, fmt.Errorf("source `%s` kind '%s' not supported",
			cueInstance.Spec.SourceRef.Name, cueInstance.Spec.SourceRef.Kind)
	}
	return source, nil
}

func (r *CueInstanceReconciler) finalize(ctx context.Context, cueInstance cuev1alpha1.CueInstance) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if cueInstance.Spec.Prune &&
		!cueInstance.Spec.Suspend &&
		cueInstance.Status.Inventory != nil &&
		cueInstance.Status.Inventory.Entries != nil {
		objects, _ := ListObjectsInInventory(cueInstance.Status.Inventory)

		impersonation := NewCueInstanceImpersonation(cueInstance, r.Client, r.StatusPoller, r.DefaultServiceAccount)
		if impersonation.CanFinalize(ctx) {
			kubeClient, _, err := impersonation.GetClient(ctx)
			if err != nil {
				return ctrl.Result{}, err
			}

			resourceManager := ssa.NewResourceManager(kubeClient, nil, ssa.Owner{
				Field: r.ControllerName,
				Group: cuev1alpha1.GroupVersion.Group,
			})

			opts := ssa.DeleteOptions{
				PropagationPolicy: metav1.DeletePropagationBackground,
				Inclusions:        resourceManager.GetOwnerLabels(cueInstance.Name, cueInstance.Namespace),
				Exclusions: map[string]string{
					fmt.Sprintf("%s/prune", cuev1alpha1.GroupVersion.Group):     cuev1alpha1.DisabledValue,
					fmt.Sprintf("%s/reconcile", cuev1alpha1.GroupVersion.Group): cuev1alpha1.DisabledValue,
				},
			}

			changeSet, err := resourceManager.DeleteAll(ctx, objects, opts)
			if err != nil {
				r.event(ctx, cueInstance, cueInstance.Status.LastAppliedRevision, events.EventSeverityError, "pruning for deleted resource failed", nil)
				// Return the error so we retry the failed garbage collection
				return ctrl.Result{}, err
			}

			if changeSet != nil && len(changeSet.Entries) > 0 {
				r.event(ctx, cueInstance, cueInstance.Status.LastAppliedRevision, events.EventSeverityInfo, changeSet.String(), nil)
			}
		} else {
			// when the account to impersonate is gone, log the stale objects and continue with the finalization
			msg := fmt.Sprintf("unable to prune objects: \n%s", ssa.FmtUnstructuredList(objects))
			log.Error(fmt.Errorf("skiping pruning, failed to find account to impersonate"), msg)
			r.event(ctx, cueInstance, cueInstance.Status.LastAppliedRevision, events.EventSeverityError, msg, nil)
		}
	}

	// Record deleted status
	r.recordReadiness(ctx, cueInstance)

	// Remove our finalizer from the list and update it
	controllerutil.RemoveFinalizer(&cueInstance, cuev1alpha1.CueInstanceFinalizer)
	if err := r.Update(ctx, &cueInstance, client.FieldOwner(r.statusManager)); err != nil {
		return ctrl.Result{}, err
	}

	// Stop reconciliation as the object is being deleted
	return ctrl.Result{}, nil
}

func (r *CueInstanceReconciler) event(ctx context.Context, cueInstance cuev1alpha1.CueInstance, revision, severity, msg string, metadata map[string]string) {
	if metadata == nil {
		metadata = map[string]string{}
	}

	if revision != "" {
		metadata[cuev1alpha1.GroupVersion.Group+"/revision"] = revision
	}

	reason := severity

	if c := apimeta.FindStatusCondition(cueInstance.Status.Conditions, meta.ReadyCondition); c != nil {
		reason = c.Reason
	}

	eventtype := "Normal"
	if severity == events.EventSeverityError {
		eventtype = "Warning"
	}

	r.EventRecorder.AnnotatedEventf(&cueInstance, metadata, eventtype, reason, msg)
}

func (r *CueInstanceReconciler) recordReadiness(ctx context.Context, cueInstance cuev1alpha1.CueInstance) {
	if r.MetricsRecorder == nil {
		return
	}
	log := ctrl.LoggerFrom(ctx)

	objRef, err := reference.GetReference(r.Scheme, &cueInstance)
	if err != nil {
		log.Error(err, "unable to record readiness metric")
		return
	}
	if rc := apimeta.FindStatusCondition(cueInstance.Status.Conditions, meta.ReadyCondition); rc != nil {
		r.MetricsRecorder.RecordCondition(*objRef, *rc, !cueInstance.DeletionTimestamp.IsZero())
	} else {
		r.MetricsRecorder.RecordCondition(*objRef, metav1.Condition{
			Type:   meta.ReadyCondition,
			Status: metav1.ConditionUnknown,
		}, !cueInstance.DeletionTimestamp.IsZero())
	}
}

func (r *CueInstanceReconciler) recordSuspension(ctx context.Context, cueInstance cuev1alpha1.CueInstance) {
	if r.MetricsRecorder == nil {
		return
	}
	log := ctrl.LoggerFrom(ctx)

	objRef, err := reference.GetReference(r.Scheme, &cueInstance)
	if err != nil {
		log.Error(err, "unable to record suspended metric")
		return
	}

	if !cueInstance.DeletionTimestamp.IsZero() {
		r.MetricsRecorder.RecordSuspend(*objRef, false)
	} else {
		r.MetricsRecorder.RecordSuspend(*objRef, cueInstance.Spec.Suspend)
	}
}

func (r *CueInstanceReconciler) patchStatus(ctx context.Context, req ctrl.Request, newStatus cuev1alpha1.CueInstanceStatus) error {
	var cueInstance cuev1alpha1.CueInstance
	if err := r.Get(ctx, req.NamespacedName, &cueInstance); err != nil {
		return err
	}

	patch := client.MergeFrom(cueInstance.DeepCopy())
	cueInstance.Status = newStatus

	return r.Status().Patch(ctx, &cueInstance, patch)
}
