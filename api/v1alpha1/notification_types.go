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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO
type GroupVersionResource struct {
	Group    string `json:"group"`
	Version  string `json:"version"`
	Resource string `json:"resource"`
}

// TODO
type NotificationCondition struct {
	Name  string `json:"name"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

// TODO
type NotificationMessage struct {
	Reason string `json:"reason"`
	Data   string `json:"data"`
}

// SynchronizationSpec defines the spec of the synchronization section of a Notification
type SynchronizationSpec struct {
	Time string `json:"time"`
}

// NotificationSpec defines the desired state of Notification
type NotificationSpec struct {
	// SynchronizationSpec defines the behavior of synchronization
	Synchronization SynchronizationSpec `json:"synchronization"`

	Watch      GroupVersionResource    `json:"watch"`
	Conditions []NotificationCondition `json:"conditions"`
	Message    NotificationMessage     `json:"message"`
}

// NotificationStatus defines the observed state of Notification
type NotificationStatus struct {

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Namespaced,categories={notifications}
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description=""

// Notification is the Schema for the notifications API
type Notification struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotificationSpec   `json:"spec,omitempty"`
	Status NotificationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NotificationList contains a list of Notification
type NotificationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Notification `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Notification{}, &NotificationList{})
}
