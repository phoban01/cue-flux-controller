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
	"context"
	"fmt"
	"strings"

	cuev1alpha1 "github.com/phoban01/cue-flux-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type CueInstanceImpersonation struct {
	workdir      string
	cueInstance  cuev1alpha1.CueInstance
	statusPoller *polling.StatusPoller
	client.Client
}

func NewCueInstanceImpersonation(
	cueInstance cuev1alpha1.CueInstance,
	kubeClient client.Client,
	statusPoller *polling.StatusPoller,
	workdir string) *CueInstanceImpersonation {
	return &CueInstanceImpersonation{
		workdir:      workdir,
		cueInstance:  cueInstance,
		statusPoller: statusPoller,
		Client:       kubeClient,
	}
}

func (ci *CueInstanceImpersonation) GetServiceAccountToken(ctx context.Context) (string, error) {
	namespacedName := types.NamespacedName{
		Namespace: ci.cueInstance.Namespace,
		Name:      ci.cueInstance.Spec.ServiceAccountName,
	}

	var serviceAccount corev1.ServiceAccount
	err := ci.Client.Get(ctx, namespacedName, &serviceAccount)
	if err != nil {
		return "", err
	}

	secretName := types.NamespacedName{
		Namespace: ci.cueInstance.Namespace,
		Name:      ci.cueInstance.Spec.ServiceAccountName,
	}

	for _, secret := range serviceAccount.Secrets {
		if strings.HasPrefix(secret.Name, fmt.Sprintf("%s-token", serviceAccount.Name)) {
			secretName.Name = secret.Name
			break
		}
	}

	var secret corev1.Secret
	err = ci.Client.Get(ctx, secretName, &secret)
	if err != nil {
		return "", err
	}

	var token string
	if data, ok := secret.Data["token"]; ok {
		token = string(data)
	} else {
		return "", fmt.Errorf("the service account secret '%s' does not containt a token", secretName.String())
	}

	return token, nil
}

// GetClient creates a controller-runtime client for talcing to a Kubernetes API server.
// If KubeConfig is set, will use the kubeconfig bytes from the Kubernetes secret.
// If ServiceAccountName is set, will use the cluster provided kubeconfig impersonating the SA.
// If --kubeconfig is set, will use the kubeconfig file at that location.
// Otherwise will assume running in cluster and use the cluster provided kubeconfig.
func (ci *CueInstanceImpersonation) GetClient(ctx context.Context) (client.Client, *polling.StatusPoller, error) {
	if ci.cueInstance.Spec.KubeConfig == nil {
		if ci.cueInstance.Spec.ServiceAccountName != "" {
			return ci.clientForServiceAccount(ctx)
		}

		return ci.Client, ci.statusPoller, nil
	}
	return ci.clientForKubeConfig(ctx)
}

func (ci *CueInstanceImpersonation) clientForServiceAccount(ctx context.Context) (client.Client, *polling.StatusPoller, error) {
	token, err := ci.GetServiceAccountToken(ctx)
	if err != nil {
		return nil, nil, err
	}
	restConfig, err := config.GetConfig()
	if err != nil {
		return nil, nil, err
	}
	restConfig.BearerToken = token
	restConfig.BearerTokenFile = "" // Clear, as it overrides BearerToken

	restMapper, err := apiutil.NewDynamicRESTMapper(restConfig)
	if err != nil {
		return nil, nil, err
	}

	client, err := client.New(restConfig, client.Options{Mapper: restMapper})
	if err != nil {
		return nil, nil, err
	}

	statusPoller := polling.NewStatusPoller(client, restMapper, nil)
	return client, statusPoller, err

}

func (ci *CueInstanceImpersonation) clientForKubeConfig(ctx context.Context) (client.Client, *polling.StatusPoller, error) {
	kubeConfigBytes, err := ci.getKubeConfig(ctx)
	if err != nil {
		return nil, nil, err
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfigBytes)
	if err != nil {
		return nil, nil, err
	}

	restMapper, err := apiutil.NewDynamicRESTMapper(restConfig)
	if err != nil {
		return nil, nil, err
	}

	client, err := client.New(restConfig, client.Options{Mapper: restMapper})
	if err != nil {
		return nil, nil, err
	}

	statusPoller := polling.NewStatusPoller(client, restMapper, nil)

	return client, statusPoller, err
}

func (ci *CueInstanceImpersonation) getKubeConfig(ctx context.Context) ([]byte, error) {
	secretName := types.NamespacedName{
		Namespace: ci.cueInstance.GetNamespace(),
		Name:      ci.cueInstance.Spec.KubeConfig.SecretRef.Name,
	}

	var secret corev1.Secret
	if err := ci.Get(ctx, secretName, &secret); err != nil {
		return nil, fmt.Errorf("unable to read KubeConfig secret '%s' error: %w", secretName.String(), err)
	}

	var kubeConfig []byte
	for k := range secret.Data {
		if k == "value" || k == "value.yaml" {
			kubeConfig = secret.Data[k]
			break
		}
	}

	if len(kubeConfig) == 0 {
		return nil, fmt.Errorf("KubeConfig secret '%s' doesn't contain a 'value' key ", secretName.String())
	}

	return kubeConfig, nil
}
