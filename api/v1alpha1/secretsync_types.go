/*
Copyright 2025.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretSyncSpec defines the desired state of SecretSync.
type SecretSyncSpec struct {
	// sourceName is the name of the source Secret to sync.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	SourceName string `json:"sourceName"`

	// sourceNamespace is the namespace of the source secret
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	SourceNamespace string `json:"sourceNamespace"`
	//
	// targetNamespaces is a list of namespaces where the source Secret should be copied to
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	TargetNamespaces []string `json:"targetNamespaces"`
}

// SecretSyncStatus defines the observed state of SecretSync.
type SecretSyncStatus struct {
	// lastSyncTime is the last time the sync operation was performed.
	LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`
	// conditions is a list of conditions that describe the current state of the SecretSync CR.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=".status.conditions[?(@.type=='Synced')].message"
// +kubebuilder:printcolumn:name="LastTransition",type=string,JSONPath=".status.conditions[?(@.type=='Synced')].lastTransitionTime"
//
// SecretSync is the Schema for the secretsyncs API.
type SecretSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretSyncSpec   `json:"spec,omitempty"`
	Status SecretSyncStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecretSyncList contains a list of SecretSync.
type SecretSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretSync `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecretSync{}, &SecretSyncList{})
}
