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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PgUserSpec defines the desired state of PgUser
type PgUserSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Name ResourceVar `json:"name"`
	// +optional
	Password ResourceVar `json:"password"`
	// +optional
	// +listType=atomic
	AccessSpecs *[]AccessSpec `json:"accessSpecs,omitempty"`
}

type Perm string

const (
	PermReadOnly  Perm = "readonly"
	PermReadWrite Perm = "readwrite"
)

// AccessSpecs defines a access request specification.
type AccessSpec struct {
	// HostCredential is the name of the PgHostCredential
	HostCredential string `json:"hostCredential"`
	// Database is the name of the PgDatabase
	Database string `json:"database"`
	// +optional
	Schema ResourceVar `json:"schema"`
	// +optional
	Reason string `json:"reason"`
	// Permission defines the access right to the database or schema
	Permission Perm `json:"permission"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="PhaseUpdated",type="string",JSONPath=".status.phaseUpdated"
// +kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.error"

// PgUser is the Schema for the pgusers API
type PgUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PgUserSpec `json:"spec,omitempty"`
	Status Status     `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PgUserList contains a list of PgUser
type PgUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PgUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PgUser{}, &PgUserList{})
}
