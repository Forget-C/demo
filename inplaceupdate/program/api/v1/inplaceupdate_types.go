/*
Copyright 2024 extreme.

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

package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type TargetReference struct {
	metav1.TypeMeta `json:",inline"`
	Name            string `json:"name"`
}

type InplaceUpdateArgs struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type ReclaimPolicyType string

const ReclaimPolicyDelete ReclaimPolicyType = "Delete"
const ReclaimPolicyRetain ReclaimPolicyType = "Retain"

type FailurePolicyType string

const FailurePolicyIgnore FailurePolicyType = "Ignore"
const FailurePolicyAbort FailurePolicyType = "Abort"

// InplaceUpdateSpec defines the desired state of InplaceUpdate
type InplaceUpdateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// TargetReference contains enough information to let you identify an workload for InplaceUpdate
	TargetReference *TargetReference `json:"targetRef"`
	// Containers defines the container to be updated
	Containers []InplaceUpdateArgs `json:"containers"`
	// RollingUpdate is a flag to indicate whether the update is rolling update
	// default is false
	// +optional
	RollingUpdate bool `json:"rollingUpdate,omitempty"`
	// The maximum number of pods that can be unavailable during update or scale.
	// Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).
	// Absolute number is calculated from percentage by rounding up by default.
	// When maxSurge > 0, absolute number is calculated from percentage by rounding down.
	// Defaults to 20%.
	// +optional
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
	// ReclaimPolicy is the policy to reclaim the resources after the update
	ReclaimPolicy ReclaimPolicyType `json:"reclaimPolicy,omitempty"`
	// Delay is the time to wait before starting the update
	// default is 0s
	// +optional
	Delay *int32 `json:"delay,omitempty"`
	// FailurePolicy is the policy to handle the failure during the update
	// default is Ignore
	// +optional
	FailurePolicy FailurePolicyType `json:"failurePolicy"`
}

type InplaceUpdateConditionType string

const InplaceUpdateConditionFailedOwner = "FailedOwnerRef"
const InplaceUpdateConditionFailedPods = "FailedPods"

type InplaceUpdateCondition struct {
	// Type of inplace update condition.
	Type InplaceUpdateConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

type InplaceUpdatePhase string

const InplaceUpdatePhasePending = "Pending"
const InplaceUpdatePhaseRunning = "Running"
const InplaceUpdatePhaseFinished = "Finished"
const InplaceUpdatePhaseFailed = "Failed"

// InplaceUpdateStatus defines the observed state of InplaceUpdate
type InplaceUpdateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Replicas is the number of pods to be updated
	Replicas int32 `json:"replicas"`
	// UpdatedReplicas is the number of pods that have been updated
	UpdatedReplicas int32 `json:"updatedReplicas"`
	// UnavailableReplicas is the number of pods that are unavailable
	UnavailableReplicas int32 `json:"unavailableReplicas"`
	// ContainerNumber is the number of containers to be updated
	ContainerNumber int32 `json:"containerNumber"`
	// UpdatedContainerNumber is the number of containers that have been updated
	UpdatedContainerNumber int32                    `json:"updatedContainerNumber"`
	Conditions             []InplaceUpdateCondition `json:"conditions,omitempty"`
	StartTime              *metav1.Time             `json:"startTime,omitempty"`
	CompletionTime         *metav1.Time             `json:"completionTime,omitempty"`
	Phase                  InplaceUpdatePhase       `json:"phase,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// InplaceUpdate is the Schema for the inplaceupdates API
type InplaceUpdate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InplaceUpdateSpec   `json:"spec,omitempty"`
	Status InplaceUpdateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// InplaceUpdateList contains a list of InplaceUpdate
type InplaceUpdateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InplaceUpdate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InplaceUpdate{}, &InplaceUpdateList{})
}
