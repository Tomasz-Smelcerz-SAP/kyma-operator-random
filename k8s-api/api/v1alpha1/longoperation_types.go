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

// LongOperationSpec defines the desired state of LongOperation
type LongOperationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	//Processing time == ConstantProcessingTime + RandomProcessingTime.
	//To get a constant processing time == X seconds, provide:
	//ConstantProcessingTime: X
	//RandomProcessingTime: 0

	//To get a random processing time in the range [X, Y] seconds, where Y > X, provide:
	//ConstantProcessingTime: X
	//RandomProcessingTime: Y - X

	//Constant processing time, in seconds.
	ConstantProcessingTime int `json:"constantProcessingTime,omitempty"`

	//Random processing time. If greater than zero, the processing time is calculated as a random number of seconds in the range: [0..RandomProcessingTime].
	RandomProcessingTime int `json:"randomProcessingTime,omitempty"`

	BigPayload string `json:"bigPayload,omitempty"`
}

// +kubebuilder:validation:Enum=Processing;Deleting;Ready;Error
type LongOperationState string

// Valid Helm States
const (
	// LongOperationStateReady signifies LongOperation is ready
	LongOperationStateReady LongOperationState = "Ready"

	// LongOperationStateProcessing signifies LongOperation is reconciling
	LongOperationStateProcessing LongOperationState = "Processing"

	// LongOperationStateError signifies an error for LongOperation
	LongOperationStateError LongOperationState = "Error"

	// LongOperationStateDeleting signifies LongOperation is being deleted
	LongOperationStateDeleting LongOperationState = "Deleting"
)

// LongOperationStatus defines the observed state of LongOperation
type LongOperationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	State              LongOperationState `json:"state,omitempty"`
	Message            string             `json:"message,omitempty"`
	Updated            string             `json:"updated,omitempty"`
	BusyUntil          string             `json:"busyUntil,omitempty"`
	ObservedGeneration int                `json:"observedGeneration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LongOperation is the Schema for the longoperations API
type LongOperation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LongOperationSpec   `json:"spec,omitempty"`
	Status LongOperationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LongOperationList contains a list of LongOperation
type LongOperationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LongOperation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LongOperation{}, &LongOperationList{})
}
