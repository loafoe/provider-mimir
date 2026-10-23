/*
Copyright 2025 The Crossplane Authors.

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
	"reflect"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupRecordingParameters defines the desired state of a GroupRecording (legacy type).
// This type exists for backward compatibility with upjet-based provider resources.
type GroupRecordingParameters struct {
	// OrgID is the tenant/organization ID. If not set, the Org ID defined in
	// the provider config will be used.
	// +optional
	OrgID *string `json:"orgId,omitempty"`

	// Namespace is the rule group namespace.
	// +kubebuilder:default="default"
	// +optional
	Namespace *string `json:"namespace,omitempty"`

	// Interval is the evaluation interval for the rule group.
	// +optional
	Interval *string `json:"interval,omitempty"`

	// QueryOffset is the duration by which to delay rule execution.
	// +optional
	QueryOffset *string `json:"queryOffset,omitempty"`

	// Rules is the list of recording rules.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Rules []RecordingRule `json:"rule"`
}

// GroupRecordingObservation represents the observed state of a GroupRecording.
type GroupRecordingObservation struct {
	// ID is the identifier for this resource.
	// +optional
	ID *string `json:"id,omitempty"`

	// OrgID is the tenant/organization ID used.
	// +optional
	OrgID *string `json:"orgId,omitempty"`

	// Namespace is the rule group namespace.
	// +optional
	Namespace *string `json:"namespace,omitempty"`

	// Interval is the evaluation interval.
	// +optional
	Interval *string `json:"interval,omitempty"`

	// QueryOffset is the query offset.
	// +optional
	QueryOffset *string `json:"queryOffset,omitempty"`

	// Rules is the list of recording rules.
	// +optional
	Rules []RecordingRule `json:"rule,omitempty"`
}

// GroupRecordingSpec defines the desired state of a GroupRecording.
type GroupRecordingSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              GroupRecordingParameters `json:"forProvider"`
}

// GroupRecordingStatus represents the observed state of a GroupRecording.
type GroupRecordingStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          GroupRecordingObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// GroupRecording is the Schema for the GroupRecording API (legacy type for backward compatibility).
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,mimir},shortName=gr
type GroupRecording struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GroupRecordingSpec   `json:"spec"`
	Status            GroupRecordingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GroupRecordingList contains a list of GroupRecording.
type GroupRecordingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GroupRecording `json:"items"`
}

// GroupRecording type metadata.
var (
	GroupRecordingKind             = reflect.TypeOf(GroupRecording{}).Name()
	GroupRecordingGroupKind        = schema.GroupKind{Group: Group, Kind: GroupRecordingKind}.String()
	GroupRecordingKindAPIVersion   = GroupRecordingKind + "." + SchemeGroupVersion.String()
	GroupRecordingGroupVersionKind = SchemeGroupVersion.WithKind(GroupRecordingKind)
)

func init() {
	SchemeBuilder.Register(&GroupRecording{}, &GroupRecordingList{})
}
