package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	. "github.com/onsi/gomega"
	cuev1alpha1 "github.com/phoban01/cue-flux-controller/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCueInstanceReconciler_BuildInstance(t *testing.T) {
	g := NewWithT(t)
	id := "builder-" + randStringRunes(5)

	err := createNamespace(id)
	g.Expect(err).NotTo(HaveOccurred(), "failed to create test namespace")

	err = createKubeConfigSecret(id)
	g.Expect(err).NotTo(HaveOccurred(), "failed to create kubeconfig secret")

	deployNamespace := "cue-build"
	err = createNamespace(deployNamespace)
	g.Expect(err).NotTo(HaveOccurred())

	artifactFile := "instance-" + randStringRunes(5)
	artifactChecksum, err := createArtifact(testServer, "testdata/multi_env", artifactFile)
	g.Expect(err).ToNot(HaveOccurred())

	repositoryName := types.NamespacedName{
		Name:      randStringRunes(5),
		Namespace: id,
	}

	err = applyGitRepository(repositoryName, artifactFile, "main/"+artifactChecksum)
	g.Expect(err).NotTo(HaveOccurred())

	cueInstanceKey := types.NamespacedName{
		Name:      "inst-" + randStringRunes(5),
		Namespace: id,
	}

	tagName := "podinfo" + randStringRunes(5)

	cueInstance := &cuev1alpha1.CueInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cueInstanceKey.Name,
			Namespace: cueInstanceKey.Namespace,
		},
		Spec: cuev1alpha1.CueInstanceSpec{
			Interval:   metav1.Duration{Duration: reconciliationInterval},
			ModuleRoot: "./testdata/multi_env",
			Path:       "cluster-01/tenant-01",
			Exprs: []string{
				"out",
			},
			Tags: []cuev1alpha1.TagVar{
				{
					Key:   "name",
					Value: tagName,
				},
				{
					Key:   "namespace",
					Value: deployNamespace,
				},
			},
			KubeConfig: &cuev1alpha1.KubeConfig{
				SecretRef: meta.LocalObjectReference{
					Name: "kubeconfig",
				},
			},
			SourceRef: cuev1alpha1.CrossNamespaceSourceReference{
				Name:      repositoryName.Name,
				Namespace: repositoryName.Namespace,
				Kind:      sourcev1.GitRepositoryKind,
			},
		},
	}

	g.Expect(k8sClient.Create(context.TODO(), cueInstance)).To(Succeed())

	g.Eventually(func() bool {
		var obj cuev1alpha1.CueInstance
		_ = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(cueInstance), &obj)
		return obj.Status.LastAppliedRevision == "main/"+artifactChecksum
	}, timeout, time.Second).Should(BeTrue())

	deployment := &appsv1.Deployment{}
	g.Expect(k8sClient.Get(context.TODO(), types.NamespacedName{
		Name:      tagName,
		Namespace: deployNamespace,
	}, deployment)).To(Succeed())

	serviceAccount := &corev1.ServiceAccount{}
	g.Expect(k8sClient.Get(context.TODO(), types.NamespacedName{
		Name:      tagName,
		Namespace: deployNamespace,
	}, serviceAccount)).To(Succeed())
}
