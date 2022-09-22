package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	. "github.com/onsi/gomega"
	cuev1alpha1 "github.com/phoban01/cue-flux-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCueInstanceReconciler_Gates(t *testing.T) {
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
	artifactChecksum, err := createArtifact(testServer, "testdata/gates", artifactFile)
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
			Interval: metav1.Duration{Duration: reconciliationInterval},
			Root:     "./testdata/gates",
			Exprs: []string{
				"out",
			},
			Gates: []cuev1alpha1.GateExpr{
				{
					Name: "deploy",
					Expr: "deployGate",
				},
			},
			Tags: []cuev1alpha1.TagVar{
				{
					Name:  "gate",
					Value: "tummy",
				},
				{
					Name:  "name",
					Value: tagName,
				},
				{
					Name:  "namespace",
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

	cm := &corev1.ConfigMap{}
	g.Expect(k8sClient.Get(context.TODO(), types.NamespacedName{
		Name:      tagName,
		Namespace: deployNamespace,
	}, cm)).ToNot(Succeed())

	cinst := &cuev1alpha1.CueInstance{}
	g.Expect(k8sClient.Get(context.TODO(), cueInstanceKey, cinst)).To(Succeed())

	patch := client.MergeFrom(cinst.DeepCopy())

	cinst.Spec.Tags[0] = cuev1alpha1.TagVar{
		Name:  "gate",
		Value: "dummy",
	}

	g.Expect(k8sClient.Patch(context.TODO(), cinst, patch)).To(Succeed())

	g.Eventually(func() bool {
		key := types.NamespacedName{
			Name:      tagName,
			Namespace: deployNamespace,
		}
		if err := k8sClient.Get(context.TODO(), key, cm); err != nil {
			return false
		}
		return true
	}, timeout, time.Second).Should(BeTrue())
}
