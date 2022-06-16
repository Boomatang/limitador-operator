/*
Copyright 2020 Red Hat.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LimitadorSpec defines the desired state of Limitador
type LimitadorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	Replicas *int `json:"replicas,omitempty"`

	// +optional
	Version *string `json:"version,omitempty"`

	// +optional
	Listener *Listener `json:"listener,omitempty"`
}

// LimitadorStatus defines the observed state of Limitador
type LimitadorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	ServiceURL string `json:"service-url,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Limitador is the Schema for the limitadors API
type Limitador struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LimitadorSpec   `json:"spec,omitempty"`
	Status LimitadorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LimitadorList contains a list of Limitador
type LimitadorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Limitador `json:"items"`
}

type Listener struct {
	// +optional
	HTTP TransportProtocol `json:"http,omitempty"`
	// +optional
	GRPC TransportProtocol `json:"grpc,omitempty"`
}

type TransportProtocol struct {
	// +optional
	Port *int32 `json:"port,omitempty"`
	// We could describe TLS within this type
}

func init() {
	SchemeBuilder.Register(&Limitador{}, &LimitadorList{})
}
