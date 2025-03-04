/*
Copyright 2024.

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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IntegrationCredentials struct {
	SecretRef v1.SecretReference `json:"secretRef,omitempty"`
}

// IntegrationWebhook TODO
type IntegrationWebhook struct {
	Url       string            `json:"url"`
	Verb      string            `json:"verb"`
	Headers   map[string]string `json:"headers,omitempty"`
	Validator string            `json:"validator,omitempty"`
}

// IntegrationSpec defines the desired state of Integration.
type IntegrationSpec struct {
	Credentials IntegrationCredentials `json:"credentials,omitempty"`

	Type    string             `json:"type"`
	Webhook IntegrationWebhook `json:"webhook,omitempty"`
}

// IntegrationStatus defines the observed state of Integration.
type IntegrationStatus struct {

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=integrations,scope=Cluster
// +kubebuilder:subresource:status

// Integration is the Schema for the integrations API.
type Integration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IntegrationSpec   `json:"spec,omitempty"`
	Status IntegrationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IntegrationList contains a list of Integration.
type IntegrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Integration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Integration{}, &IntegrationList{})
}
