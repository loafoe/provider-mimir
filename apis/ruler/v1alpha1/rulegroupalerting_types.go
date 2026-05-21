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

// RuleGroupAlertingParameters defines the desired state of a RuleGroupAlerting.
type RuleGroupAlertingParameters struct {
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

	// SourceTenants allows aggregating data from multiple tenants.
	// +optional
	SourceTenants []string `json:"sourceTenants,omitempty"`

	// Rules is the list of alerting rules.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Rules []AlertingRule `json:"rule"`
}

// AlertingRule defines an alerting rule.
type AlertingRule struct {
	// Alert is the name of the alert.
	// +kubebuilder:validation:Required
	Alert string `json:"alert"`

	// Expr is the PromQL expression to evaluate.
	// +kubebuilder:validation:Required
	Expr string `json:"expr"`

	// For is the duration for which the condition must be true before firing.
	// +optional
	For *string `json:"for,omitempty"`

	// KeepFiringFor is how long an alert continues firing after the condition clears.
	// +optional
	KeepFiringFor *string `json:"keepFiringFor,omitempty"`

	// Labels to add or overwrite for each alert.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations to add to each alert.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// RuleGroupAlertingObservation represents the observed state of a RuleGroupAlerting.
type RuleGroupAlertingObservation struct {
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

// RuleGroupAlertingSpec defines the desired state of a RuleGroupAlerting.
type RuleGroupAlertingSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              RuleGroupAlertingParameters `json:"forProvider"`
}

// RuleGroupAlertingStatus represents the observed state of a RuleGroupAlerting.
type RuleGroupAlertingStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RuleGroupAlertingObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// RuleGroupAlerting is the Schema for the RuleGroupAlerting API.
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,mimir}
type RuleGroupAlerting struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RuleGroupAlertingSpec   `json:"spec"`
	Status            RuleGroupAlertingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RuleGroupAlertingList contains a list of RuleGroupAlerting.
type RuleGroupAlertingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RuleGroupAlerting `json:"items"`
}

// RuleGroupAlerting type metadata.
var (
	RuleGroupAlertingKind             = reflect.TypeOf(RuleGroupAlerting{}).Name()
	RuleGroupAlertingGroupKind        = schema.GroupKind{Group: Group, Kind: RuleGroupAlertingKind}.String()
	RuleGroupAlertingKindAPIVersion   = RuleGroupAlertingKind + "." + SchemeGroupVersion.String()
	RuleGroupAlertingGroupVersionKind = SchemeGroupVersion.WithKind(RuleGroupAlertingKind)
)

func init() {
	SchemeBuilder.Register(&RuleGroupAlerting{}, &RuleGroupAlertingList{})
}
