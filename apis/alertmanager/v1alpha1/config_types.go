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

// ConfigParameters defines the desired state of an AlertmanagerConfig.
type ConfigParameters struct {
	// OrgID is the tenant/organization ID. If not set, the Org ID defined in
	// the provider config will be used.
	// +optional
	OrgID *string `json:"orgId,omitempty"`

	// Global contains global alertmanager configuration.
	// +optional
	Global *GlobalConfig `json:"global,omitempty"`

	// Route defines the routing tree.
	// +optional
	Route *RouteConfig `json:"route,omitempty"`

	// Receivers is the list of notification receivers.
	// +optional
	Receivers []ReceiverConfig `json:"receiver,omitempty"`

	// InhibitRules is the list of inhibition rules.
	// +optional
	InhibitRules []InhibitRuleConfig `json:"inhibitRule,omitempty"`

	// TimeIntervals is the list of mute time intervals.
	// +optional
	TimeIntervals []TimeIntervalConfig `json:"timeInterval,omitempty"`

	// Templates is a list of template file names to use.
	// +optional
	Templates []string `json:"templates,omitempty"`

	// TemplatesFiles is a map of template names to template content.
	// +optional
	TemplatesFiles map[string]string `json:"templatesFiles,omitempty"`
}

// GlobalConfig contains global alertmanager configuration.
type GlobalConfig struct {
	// ResolveTimeout is the time after which an alert is declared resolved.
	// +optional
	ResolveTimeout *string `json:"resolveTimeout,omitempty"`

	// SMTPSmarthost is the default SMTP smarthost for notifications.
	// +optional
	SMTPSmarthost *string `json:"smtpSmarthost,omitempty"`

	// SMTPFrom is the default SMTP from address.
	// +optional
	SMTPFrom *string `json:"smtpFrom,omitempty"`

	// SMTPAuthUsername is the SMTP AUTH username.
	// +optional
	SMTPAuthUsername *string `json:"smtpAuthUsername,omitempty"`

	// SMTPAuthPasswordSecretRef is a reference to a secret containing the SMTP AUTH password.
	// +optional
	SMTPAuthPasswordSecretRef *xpv1.SecretKeySelector `json:"smtpAuthPasswordSecretRef,omitempty"`

	// SMTPAuthIdentity is the SMTP AUTH identity.
	// +optional
	SMTPAuthIdentity *string `json:"smtpAuthIdentity,omitempty"`

	// SMTPRequireTLS requires TLS for SMTP connections.
	// +optional
	SMTPRequireTLS *bool `json:"smtpRequireTls,omitempty"`

	// SlackAPIURLSecretRef is a reference to a secret containing the Slack API URL.
	// +optional
	SlackAPIURLSecretRef *xpv1.SecretKeySelector `json:"slackApiUrlSecretRef,omitempty"`

	// PagerdutyURL is the PagerDuty API URL.
	// +optional
	PagerdutyURL *string `json:"pagerdutyUrl,omitempty"`

	// OpsGenieAPIURL is the OpsGenie API URL.
	// +optional
	OpsGenieAPIURL *string `json:"opsgenieApiUrl,omitempty"`

	// OpsGenieAPIKeySecretRef is a reference to a secret containing the OpsGenie API key.
	// +optional
	OpsGenieAPIKeySecretRef *xpv1.SecretKeySelector `json:"opsgenieApiKeySecretRef,omitempty"`
}

// RouteConfig defines a routing tree node.
type RouteConfig struct {
	// Receiver is the name of the receiver to use.
	// +optional
	Receiver *string `json:"receiver,omitempty"`

	// GroupBy is a list of labels to group alerts by.
	// +optional
	GroupBy []string `json:"groupBy,omitempty"`

	// GroupWait is how long to wait before sending the initial notification.
	// +optional
	GroupWait *string `json:"groupWait,omitempty"`

	// GroupInterval is how long to wait before sending updated notifications.
	// +optional
	GroupInterval *string `json:"groupInterval,omitempty"`

	// RepeatInterval is how long to wait before re-sending a notification.
	// +optional
	RepeatInterval *string `json:"repeatInterval,omitempty"`

	// Continue indicates whether to continue matching subsequent sibling nodes.
	// +optional
	Continue *bool `json:"continue,omitempty"`

	// Matchers is a list of matchers that an alert must fulfill.
	// +optional
	Matchers []string `json:"matchers,omitempty"`

	// MuteTimeIntervals is a list of mute time interval names to apply.
	// +optional
	MuteTimeIntervals []string `json:"muteTimeIntervals,omitempty"`

	// ActiveTimeIntervals is a list of active time interval names.
	// +optional
	ActiveTimeIntervals []string `json:"activeTimeIntervals,omitempty"`

	// ChildRoute contains nested routes (JSON representation for nested routes).
	// Use RouteConfigJSON type for nested routes to avoid infinite recursion.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	ChildRoute []ChildRouteConfig `json:"childRoute,omitempty"`
}

// ChildRouteConfig defines a child routing node (limited nesting to avoid infinite recursion).
type ChildRouteConfig struct {
	// Receiver is the name of the receiver to use.
	// +optional
	Receiver *string `json:"receiver,omitempty"`

	// GroupBy is a list of labels to group alerts by.
	// +optional
	GroupBy []string `json:"groupBy,omitempty"`

	// GroupWait is how long to wait before sending the initial notification.
	// +optional
	GroupWait *string `json:"groupWait,omitempty"`

	// GroupInterval is how long to wait before sending updated notifications.
	// +optional
	GroupInterval *string `json:"groupInterval,omitempty"`

	// RepeatInterval is how long to wait before re-sending a notification.
	// +optional
	RepeatInterval *string `json:"repeatInterval,omitempty"`

	// Continue indicates whether to continue matching subsequent sibling nodes.
	// +optional
	Continue *bool `json:"continue,omitempty"`

	// Matchers is a list of matchers that an alert must fulfill.
	// +optional
	Matchers []string `json:"matchers,omitempty"`

	// MuteTimeIntervals is a list of mute time interval names to apply.
	// +optional
	MuteTimeIntervals []string `json:"muteTimeIntervals,omitempty"`

	// ActiveTimeIntervals is a list of active time interval names.
	// +optional
	ActiveTimeIntervals []string `json:"activeTimeIntervals,omitempty"`
}

// ReceiverConfig defines a notification receiver.
type ReceiverConfig struct {
	// Name is the unique name of the receiver.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// EmailConfigs is a list of email notification configurations.
	// +optional
	EmailConfigs []EmailConfigSpec `json:"emailConfig,omitempty"`

	// SlackConfigs is a list of Slack notification configurations.
	// +optional
	SlackConfigs []SlackConfigSpec `json:"slackConfig,omitempty"`

	// PagerdutyConfigs is a list of PagerDuty notification configurations.
	// +optional
	PagerdutyConfigs []PagerdutyConfigSpec `json:"pagerdutyConfig,omitempty"`

	// WebhookConfigs is a list of webhook notification configurations.
	// +optional
	WebhookConfigs []WebhookConfigSpec `json:"webhookConfig,omitempty"`

	// OpsGenieConfigs is a list of OpsGenie notification configurations.
	// +optional
	OpsGenieConfigs []OpsGenieConfigSpec `json:"opsgenieConfig,omitempty"`
}

// EmailConfigSpec defines email notification configuration.
type EmailConfigSpec struct {
	// To is the email address to send to.
	// +optional
	To *string `json:"to,omitempty"`

	// From is the sender address.
	// +optional
	From *string `json:"from,omitempty"`

	// Smarthost is the SMTP server address.
	// +optional
	Smarthost *string `json:"smarthost,omitempty"`

	// AuthUsername is the SMTP AUTH username.
	// +optional
	AuthUsername *string `json:"authUsername,omitempty"`

	// AuthPasswordSecretRef is a reference to a secret containing the SMTP AUTH password.
	// +optional
	AuthPasswordSecretRef *xpv1.SecretKeySelector `json:"authPasswordSecretRef,omitempty"`

	// HTML is the HTML body of the email.
	// +optional
	HTML *string `json:"html,omitempty"`

	// Text is the text body of the email.
	// +optional
	Text *string `json:"text,omitempty"`

	// RequireTLS requires TLS for the connection.
	// +optional
	RequireTLS *bool `json:"requireTls,omitempty"`

	// SendResolved indicates whether to send resolved notifications.
	// +optional
	SendResolved *bool `json:"sendResolved,omitempty"`
}

// SlackConfigSpec defines Slack notification configuration.
type SlackConfigSpec struct {
	// APIURLSecretRef is a reference to a secret containing the Slack API URL.
	// +optional
	APIURLSecretRef *xpv1.SecretKeySelector `json:"apiUrlSecretRef,omitempty"`

	// Channel is the Slack channel to send to.
	// +optional
	Channel *string `json:"channel,omitempty"`

	// Username is the bot username.
	// +optional
	Username *string `json:"username,omitempty"`

	// IconEmoji is the emoji to use as the icon.
	// +optional
	IconEmoji *string `json:"iconEmoji,omitempty"`

	// IconURL is the URL of the icon to use.
	// +optional
	IconURL *string `json:"iconUrl,omitempty"`

	// Title is the message title.
	// +optional
	Title *string `json:"title,omitempty"`

	// Text is the message text.
	// +optional
	Text *string `json:"text,omitempty"`

	// SendResolved indicates whether to send resolved notifications.
	// +optional
	SendResolved *bool `json:"sendResolved,omitempty"`
}

// PagerdutyConfigSpec defines PagerDuty notification configuration.
type PagerdutyConfigSpec struct {
	// ServiceKeySecretRef is a reference to a secret containing the PagerDuty service key.
	// +optional
	ServiceKeySecretRef *xpv1.SecretKeySelector `json:"serviceKeySecretRef,omitempty"`

	// RoutingKeySecretRef is a reference to a secret containing the PagerDuty routing key.
	// +optional
	RoutingKeySecretRef *xpv1.SecretKeySelector `json:"routingKeySecretRef,omitempty"`

	// URL is the PagerDuty API URL.
	// +optional
	URL *string `json:"url,omitempty"`

	// Description is the description of the incident.
	// +optional
	Description *string `json:"description,omitempty"`

	// Severity is the severity of the incident.
	// +optional
	Severity *string `json:"severity,omitempty"`

	// SendResolved indicates whether to send resolved notifications.
	// +optional
	SendResolved *bool `json:"sendResolved,omitempty"`
}

// WebhookConfigSpec defines webhook notification configuration.
type WebhookConfigSpec struct {
	// URL is the webhook URL.
	// +optional
	URL *string `json:"url,omitempty"`

	// SendResolved indicates whether to send resolved notifications.
	// +optional
	SendResolved *bool `json:"sendResolved,omitempty"`

	// HTTPConfig contains HTTP client configuration.
	// +optional
	HTTPConfig *HTTPConfigSpec `json:"httpConfig,omitempty"`
}

// HTTPConfigSpec defines HTTP client configuration.
type HTTPConfigSpec struct {
	// BasicAuth contains basic authentication credentials.
	// +optional
	BasicAuth *HTTPBasicAuthSpec `json:"basicAuth,omitempty"`

	// BearerTokenSecretRef is a reference to a secret containing the bearer token.
	// +optional
	BearerTokenSecretRef *xpv1.SecretKeySelector `json:"bearerTokenSecretRef,omitempty"`
}

// HTTPBasicAuthSpec defines HTTP basic authentication.
type HTTPBasicAuthSpec struct {
	// Username is the username for basic auth.
	// +optional
	Username *string `json:"username,omitempty"`

	// PasswordSecretRef is a reference to a secret containing the password.
	// +optional
	PasswordSecretRef *xpv1.SecretKeySelector `json:"passwordSecretRef,omitempty"`
}

// OpsGenieConfigSpec defines OpsGenie notification configuration.
type OpsGenieConfigSpec struct {
	// APIKeySecretRef is a reference to a secret containing the OpsGenie API key.
	// +optional
	APIKeySecretRef *xpv1.SecretKeySelector `json:"apiKeySecretRef,omitempty"`

	// APIURL is the OpsGenie API URL.
	// +optional
	APIURL *string `json:"apiUrl,omitempty"`

	// Message is the alert message.
	// +optional
	Message *string `json:"message,omitempty"`

	// Priority is the alert priority.
	// +optional
	Priority *string `json:"priority,omitempty"`

	// SendResolved indicates whether to send resolved notifications.
	// +optional
	SendResolved *bool `json:"sendResolved,omitempty"`
}

// InhibitRuleConfig defines an inhibition rule.
type InhibitRuleConfig struct {
	// SourceMatchers is a list of matchers for source alerts.
	// +optional
	SourceMatchers []string `json:"sourceMatchers,omitempty"`

	// TargetMatchers is a list of matchers for target alerts.
	// +optional
	TargetMatchers []string `json:"targetMatchers,omitempty"`

	// Equal is a list of labels that must have equal values.
	// +optional
	Equal []string `json:"equal,omitempty"`
}

// TimeIntervalConfig defines a mute time interval.
type TimeIntervalConfig struct {
	// Name is the unique name of the time interval.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// TimeIntervals is a list of time interval definitions.
	// +optional
	TimeIntervals []TimeIntervalSpec `json:"timeIntervals,omitempty"`
}

// TimeIntervalSpec defines a time interval.
type TimeIntervalSpec struct {
	// Times is a list of time ranges.
	// +optional
	Times []TimeRangeSpec `json:"times,omitempty"`

	// Weekdays is a list of weekdays.
	// +optional
	Weekdays []string `json:"weekdays,omitempty"`

	// DaysOfMonth is a list of days of the month.
	// +optional
	DaysOfMonth []string `json:"daysOfMonth,omitempty"`

	// Months is a list of months.
	// +optional
	Months []string `json:"months,omitempty"`

	// Years is a list of years.
	// +optional
	Years []string `json:"years,omitempty"`
}

// TimeRangeSpec defines a time range.
type TimeRangeSpec struct {
	// StartTime is the start time (HH:MM format).
	// +optional
	StartTime *string `json:"startTime,omitempty"`

	// EndTime is the end time (HH:MM format).
	// +optional
	EndTime *string `json:"endTime,omitempty"`
}

// ConfigObservation represents the observed state of an AlertmanagerConfig.
type ConfigObservation struct {
	// OrgID is the tenant/organization ID used.
	// +optional
	OrgID *string `json:"orgId,omitempty"`
}

// ConfigSpec defines the desired state of an AlertmanagerConfig.
type ConfigSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ConfigParameters `json:"forProvider"`
}

// ConfigStatus represents the observed state of an AlertmanagerConfig.
type ConfigStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ConfigObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// Config is the Schema for the AlertmanagerConfig API.
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,mimir}
type Config struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ConfigSpec   `json:"spec"`
	Status            ConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ConfigList contains a list of Config.
type ConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Config `json:"items"`
}

// Config type metadata.
var (
	ConfigKind             = reflect.TypeOf(Config{}).Name()
	ConfigGroupKind        = schema.GroupKind{Group: Group, Kind: ConfigKind}.String()
	ConfigKindAPIVersion   = ConfigKind + "." + SchemeGroupVersion.String()
	ConfigGroupVersionKind = SchemeGroupVersion.WithKind(ConfigKind)
)

func init() {
	SchemeBuilder.Register(&Config{}, &ConfigList{})
}
