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

package v1alpha1

import (
	"reflect"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RuleGroupRecordingParameters defines the desired state of a RuleGroupRecording.
type RuleGroupRecordingParameters struct {
	// OrgID is the tenant/organization ID. If not set, the Org ID defined in
	// the provider config will be used.
	// +optional
	OrgID *string `json:"orgId,omitempty"`

	// Namespace is the rule group namespace.
	// +kubebuilder:default="default"
	// +optional
	Namespace *string `json:"namespace,omitempty"`

	// Name is the rule group name.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Interval is the evaluation interval for the rule group.
	// +optional
	Interval *string `json:"interval,omitempty"`

	// QueryOffset is the duration by which to delay rule execution.
	// +optional
	QueryOffset *string `json:"queryOffset,omitempty"`

	// EvaluationDelay is deprecated, use QueryOffset instead.
	// +optional
	EvaluationDelay *string `json:"evaluationDelay,omitempty"`

	// SourceTenants allows aggregating data from multiple tenants.
	// +optional
	SourceTenants []string `json:"sourceTenants,omitempty"`

	// Rules is the list of recording rules.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Rules []RecordingRule `json:"rule"`
}

// RecordingRule defines a recording rule.
type RecordingRule struct {
	// Record is the name of the time series to output to.
	// +kubebuilder:validation:Required
	Record string `json:"record"`

	// Expr is the PromQL expression to evaluate.
	// +kubebuilder:validation:Required
	Expr string `json:"expr"`

	// Labels to add or overwrite before storing the result.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// RuleGroupRecordingObservation represents the observed state of a RuleGroupRecording.
type RuleGroupRecordingObservation struct {
	// OrgID is the tenant/organization ID used.
	// +optional
	OrgID *string `json:"orgId,omitempty"`

	// Namespace is the rule group namespace.
	// +optional
	Namespace *string `json:"namespace,omitempty"`

	// Name is the rule group name.
	// +optional
	Name *string `json:"name,omitempty"`
}

// RuleGroupRecordingSpec defines the desired state of a RuleGroupRecording.
type RuleGroupRecordingSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              RuleGroupRecordingParameters `json:"forProvider"`
}

// RuleGroupRecordingStatus represents the observed state of a RuleGroupRecording.
type RuleGroupRecordingStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RuleGroupRecordingObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// RuleGroupRecording is the Schema for the RuleGroupRecording API.
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,mimir}
type RuleGroupRecording struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RuleGroupRecordingSpec   `json:"spec"`
	Status            RuleGroupRecordingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RuleGroupRecordingList contains a list of RuleGroupRecording.
type RuleGroupRecordingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RuleGroupRecording `json:"items"`
}

// RuleGroupRecording type metadata.
var (
	RuleGroupRecordingKind             = reflect.TypeOf(RuleGroupRecording{}).Name()
	RuleGroupRecordingGroupKind        = schema.GroupKind{Group: Group, Kind: RuleGroupRecordingKind}.String()
	RuleGroupRecordingKindAPIVersion   = RuleGroupRecordingKind + "." + SchemeGroupVersion.String()
	RuleGroupRecordingGroupVersionKind = SchemeGroupVersion.WithKind(RuleGroupRecordingKind)
)

func init() {
	SchemeBuilder.Register(&RuleGroupRecording{}, &RuleGroupRecordingList{})
}
