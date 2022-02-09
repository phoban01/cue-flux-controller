/*
Copyright 2022.

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

	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/runtime/acl"
	"github.com/fluxcd/pkg/runtime/events"
	"github.com/fluxcd/pkg/ssa"
	"github.com/fluxcd/pkg/untar"
	"github.com/hashicorp/go-retryablehttp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/reference"

	cuev1alpha1 "cue-flux-controller.git/api/v1alpha1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
)

// CueInstanceReconciler reconciles a CueInstance object
type CueInstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cue.contrib.flux.io,resources=cueinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cue.contrib.flux.io,resources=cueinstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cue.contrib.flux.io,resources=cueinstances/finalizers,verbs=update

// SetupWithManager sets up the controller with the Manager.
func (r *CueInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cuev1alpha1.CueInstance{}).
		Complete(r)
}

// Reconcile is the main reconciler loop.
func (r *CueInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

func (r *CueInstanceReconciler) reconcile(ctx context.Context, obj *cuev1alpha1.CueInstance) error {
	return nil
}

func (r *CueInstanceReconciler) build(ctx context.Context, dir string, tags map[string]string) ([]byte, error) {
	return nil, nil
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
		dName := types.NamespacedName(d)
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
	// Record deleted status
	r.recordReadiness(ctx, cueInstance)

	// Remove our finalizer from the list and update it
	controllerutil.RemoveFinalizer(&cueInstance, cuev1alpha1.CueInstanceFinalizer)
	if err := r.Update(ctx, &cueInstance); err != nil {
		return ctrl.Result{}, err
	}

	// Stop reconciliation as the object is being deleted
	return ctrl.Result{}, nil
}

func (r *CueInstanceReconciler) event(ctx context.Context, cueInstance cuev1alpha1.CueInstance, revision, severity, msg string, metadata map[string]string) {
	log := ctrl.LoggerFrom(ctx)

	if r.EventRecorder != nil {
		annotations := map[string]string{
			cuev1alpha1.GroupVersion.Group + "/revision": revision,
		}

		eventtype := "Normal"
		if severity == events.EventSeverityError {
			eventtype = "Warning"
		}

		r.EventRecorder.AnnotatedEventf(&cueInstance, annotations, eventtype, severity, msg)
	}

	if r.ExternalEventRecorder != nil {
		objRef, err := reference.GetReference(r.Scheme, &cueInstance)
		if err != nil {
			log.Error(err, "unable to send event")
			return
		}
		if metadata == nil {
			metadata = map[string]string{}
		}
		if revision != "" {
			metadata["revision"] = revision
		}

		reason := severity
		if c := apimeta.FindStatusCondition(cueInstance.Status.Conditions, meta.ReadyCondition); c != nil {
			reason = c.Reason
		}

		if err := r.ExternalEventRecorder.Eventf(*objRef, metadata, severity, reason, msg); err != nil {
			log.Error(err, "unable to send event")
			return
		}
	}
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
