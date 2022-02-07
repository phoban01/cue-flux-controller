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

package v1alpha1

import (
	"github.com/fluxcd/pkg/apis/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CueInstanceSpec defines the desired state of CueInstance
type CueInstanceSpec struct {
	// +required
	Interval metav1.Duration `json:"interval"`
	// +required
	SourceRef CrossNamespaceSourceReference `json:"sourceRef"`
	// +optional
	Inject []InjectItem `json:"inject,omitempty"`

	// +optional
	Path string `json:"path"`

	// Prune enables garbage collection.
	// +required
	Prune bool `json:"prune"`
}

type InjectItem struct {
	// +required
	Name string `json:"name"`
	// +required
	Value string `json:"value"`
}

// CueInstanceStatus defines the observed state of CueInstance
type CueInstanceStatus struct {
	meta.ReconcileRequestStatus `json:",inline"`

	// ObservedGeneration is the last reconciled generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// The last successfully applied revision.
	// The revision format for Git sources is <branch|tag>/<commit-sha>.
	// +optional
	LastAppliedRevision string `json:"lastAppliedRevision,omitempty"`

	// LastAttemptedRevision is the revision of the last reconciliation attempt.
	// +optional
	LastAttemptedRevision string `json:"lastAttemptedRevision,omitempty"`

	// Inventory contains the list of Kubernetes resource object references that have been successfully applied.
	// +optional
	Inventory *ResourceInventory `json:"inventory,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CueInstance is the Schema for the cueinstances API
type CueInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CueInstanceSpec   `json:"spec,omitempty"`
	Status CueInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CueInstanceList contains a list of CueInstance
type CueInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CueInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CueInstance{}, &CueInstanceList{})
}
