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

package mimir

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config contains configuration for creating a Mimir client.
type Config struct {
	// URI is the base URL of the Mimir instance.
	URI string
	// RulerURI is the URL for the ruler component (optional, defaults to URI).
	RulerURI string
	// AlertmanagerURI is the URL for the alertmanager component (optional, defaults to URI).
	AlertmanagerURI string
	// OrgID is the tenant/organization ID for multi-tenancy.
	OrgID string
	// Token for bearer authentication (optional if using basic auth).
	Token string
	// Username for basic auth (optional if using token).
	Username string
	// Password for basic auth (optional if using token).
	Password string
	// Insecure skips TLS verification.
	Insecure bool
	// CA is the CA certificate (filepath or inline PEM).
	CA string
	// Cert is the client certificate (filepath or inline PEM).
	Cert string
	// Key is the client key (filepath or inline PEM).
	Key string
	// Headers are additional headers to send with requests.
	Headers map[string]string
	// Timeout is the HTTP client timeout.
	Timeout time.Duration
}

// Client is a Mimir API client.
type Client struct {
	httpClient      *http.Client
	uri             string
	rulerURI        string
	alertmanagerURI string
	orgID           string
	token           string
	username        string
	password        string
	headers         map[string]string
}

// NewClient creates a new Mimir API client.
func NewClient(cfg Config) (*Client, error) {
	if cfg.URI == "" {
		return nil, fmt.Errorf("mimir URI is required")
	}

	uri := strings.TrimSuffix(cfg.URI, "/")
	rulerURI := strings.TrimSuffix(cfg.RulerURI, "/")
	alertmanagerURI := strings.TrimSuffix(cfg.AlertmanagerURI, "/")

	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.Insecure, //nolint:gosec
	}

	if cfg.Cert != "" && cfg.Key != "" {
		var cert tls.Certificate
		var err error
		if strings.HasPrefix(cfg.Cert, "-----BEGIN") && strings.HasPrefix(cfg.Key, "-----BEGIN") {
			cert, err = tls.X509KeyPair([]byte(cfg.Cert), []byte(cfg.Key))
		} else {
			cert, err = tls.LoadX509KeyPair(cfg.Cert, cfg.Key)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if cfg.CA != "" {
		var caCert []byte
		var err error
		if strings.HasPrefix(cfg.CA, "-----BEGIN") {
			caCert = []byte(cfg.CA)
		} else {
			caCert, err = io.ReadAll(strings.NewReader(cfg.CA))
			if err != nil {
				return nil, fmt.Errorf("failed to read CA certificate: %w", err)
			}
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		Proxy:           http.ProxyFromEnvironment,
	}

	return &Client{
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		uri:             uri,
		rulerURI:        rulerURI,
		alertmanagerURI: alertmanagerURI,
		orgID:           cfg.OrgID,
		token:           cfg.Token,
		username:        cfg.Username,
		password:        cfg.Password,
		headers:         cfg.Headers,
	}, nil
}

// sendRequest performs an HTTP request to the Mimir API.
func (c *Client) sendRequest(ctx context.Context, component, method, path, data string, extraHeaders map[string]string) (string, error) {
	var fullURI string

	switch component {
	case "ruler":
		if c.rulerURI != "" {
			fullURI = c.rulerURI + path
		} else {
			fullURI = c.uri + path
		}
	case "alertmanager":
		if c.alertmanagerURI != "" {
			fullURI = c.alertmanagerURI + path
		} else {
			fullURI = c.uri + path
		}
	default:
		fullURI = c.uri + path
	}

	var reqBody io.Reader
	if data != "" {
		reqBody = bytes.NewBufferString(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURI, reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	} else if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	if c.orgID != "" && req.Header.Get("X-Scope-OrgID") == "" {
		req.Header.Set("X-Scope-OrgID", c.orgID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return string(body), fmt.Errorf("unexpected response code '%d': %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

// IsNotFound checks if an error indicates the resource was not found.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "response code '404'") ||
		strings.Contains(err.Error(), "does not exist")
}

// AlertmanagerUserConfig represents the top-level alertmanager config structure.
type AlertmanagerUserConfig struct {
	TemplateFiles      map[string]string `yaml:"template_files,omitempty"`
	AlertmanagerConfig string            `yaml:"alertmanager_config"`
}

// AlertmanagerConfig represents the alertmanager configuration.
type AlertmanagerConfig struct {
	Global            *GlobalConfig       `yaml:"global,omitempty"`
	Route             *Route              `yaml:"route,omitempty"`
	Receivers         []Receiver          `yaml:"receivers,omitempty"`
	InhibitRules      []InhibitRule       `yaml:"inhibit_rules,omitempty"`
	MuteTimeIntervals []MuteTimeInterval  `yaml:"mute_time_intervals,omitempty"`
	Templates         []string            `yaml:"templates,omitempty"`
}

// GlobalConfig represents alertmanager global configuration.
type GlobalConfig struct {
	ResolveTimeout   string `yaml:"resolve_timeout,omitempty"`
	SMTPSmarthost    string `yaml:"smtp_smarthost,omitempty"`
	SMTPFrom         string `yaml:"smtp_from,omitempty"`
	SMTPAuthUsername string `yaml:"smtp_auth_username,omitempty"`
	SMTPAuthPassword string `yaml:"smtp_auth_password,omitempty"`
	SMTPAuthIdentity string `yaml:"smtp_auth_identity,omitempty"`
	SMTPRequireTLS   *bool  `yaml:"smtp_require_tls,omitempty"`
	SlackAPIURL      string `yaml:"slack_api_url,omitempty"`
	PagerdutyURL     string `yaml:"pagerduty_url,omitempty"`
	OpsGenieAPIURL   string `yaml:"opsgenie_api_url,omitempty"`
	OpsGenieAPIKey   string `yaml:"opsgenie_api_key,omitempty"`
}

// Route represents an alertmanager routing tree node.
type Route struct {
	Receiver            string   `yaml:"receiver,omitempty"`
	GroupBy             []string `yaml:"group_by,omitempty"`
	GroupWait           string   `yaml:"group_wait,omitempty"`
	GroupInterval       string   `yaml:"group_interval,omitempty"`
	RepeatInterval      string   `yaml:"repeat_interval,omitempty"`
	Continue            bool     `yaml:"continue,omitempty"`
	Matchers            []string `yaml:"matchers,omitempty"`
	MuteTimeIntervals   []string `yaml:"mute_time_intervals,omitempty"`
	ActiveTimeIntervals []string `yaml:"active_time_intervals,omitempty"`
	Routes              []Route  `yaml:"routes,omitempty"`
}

// Receiver represents an alertmanager notification receiver.
type Receiver struct {
	Name             string            `yaml:"name"`
	EmailConfigs     []EmailConfig     `yaml:"email_configs,omitempty"`
	SlackConfigs     []SlackConfig     `yaml:"slack_configs,omitempty"`
	PagerdutyConfigs []PagerdutyConfig `yaml:"pagerduty_configs,omitempty"`
	WebhookConfigs   []WebhookConfig   `yaml:"webhook_configs,omitempty"`
	OpsGenieConfigs  []OpsGenieConfig  `yaml:"opsgenie_configs,omitempty"`
}

// EmailConfig represents email notification configuration.
type EmailConfig struct {
	To           string            `yaml:"to,omitempty"`
	From         string            `yaml:"from,omitempty"`
	Smarthost    string            `yaml:"smarthost,omitempty"`
	AuthUsername string            `yaml:"auth_username,omitempty"`
	AuthPassword string            `yaml:"auth_password,omitempty"`
	Headers      map[string]string `yaml:"headers,omitempty"`
	HTML         string            `yaml:"html,omitempty"`
	Text         string            `yaml:"text,omitempty"`
	RequireTLS   *bool             `yaml:"require_tls,omitempty"`
	SendResolved bool              `yaml:"send_resolved,omitempty"`
}

// SlackConfig represents Slack notification configuration.
type SlackConfig struct {
	APIURL       string `yaml:"api_url,omitempty"`
	Channel      string `yaml:"channel,omitempty"`
	Username     string `yaml:"username,omitempty"`
	IconEmoji    string `yaml:"icon_emoji,omitempty"`
	IconURL      string `yaml:"icon_url,omitempty"`
	Title        string `yaml:"title,omitempty"`
	Text         string `yaml:"text,omitempty"`
	SendResolved bool   `yaml:"send_resolved,omitempty"`
}

// PagerdutyConfig represents PagerDuty notification configuration.
type PagerdutyConfig struct {
	ServiceKey  string `yaml:"service_key,omitempty"`
	RoutingKey  string `yaml:"routing_key,omitempty"`
	URL         string `yaml:"url,omitempty"`
	Description string `yaml:"description,omitempty"`
	Severity    string `yaml:"severity,omitempty"`
	SendResolved bool  `yaml:"send_resolved,omitempty"`
}

// WebhookConfig represents webhook notification configuration.
type WebhookConfig struct {
	URL          string `yaml:"url,omitempty"`
	SendResolved bool   `yaml:"send_resolved,omitempty"`
	HTTPConfig   *HTTPClientConfig `yaml:"http_config,omitempty"`
}

// OpsGenieConfig represents OpsGenie notification configuration.
type OpsGenieConfig struct {
	APIKey       string `yaml:"api_key,omitempty"`
	APIURL       string `yaml:"api_url,omitempty"`
	Message      string `yaml:"message,omitempty"`
	Priority     string `yaml:"priority,omitempty"`
	SendResolved bool   `yaml:"send_resolved,omitempty"`
}

// HTTPClientConfig represents HTTP client configuration.
type HTTPClientConfig struct {
	BasicAuth *BasicAuth `yaml:"basic_auth,omitempty"`
	BearerToken string   `yaml:"bearer_token,omitempty"`
}

// BasicAuth represents HTTP basic authentication.
type BasicAuth struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// InhibitRule represents an alertmanager inhibit rule.
type InhibitRule struct {
	SourceMatchers []string `yaml:"source_matchers,omitempty"`
	TargetMatchers []string `yaml:"target_matchers,omitempty"`
	Equal          []string `yaml:"equal,omitempty"`
}

// MuteTimeInterval represents an alertmanager mute time interval.
type MuteTimeInterval struct {
	Name          string         `yaml:"name"`
	TimeIntervals []TimeInterval `yaml:"time_intervals,omitempty"`
}

// TimeInterval represents a time interval.
type TimeInterval struct {
	Times       []TimeRange `yaml:"times,omitempty"`
	Weekdays    []string    `yaml:"weekdays,omitempty"`
	DaysOfMonth []string    `yaml:"days_of_month,omitempty"`
	Months      []string    `yaml:"months,omitempty"`
	Years       []string    `yaml:"years,omitempty"`
}

// TimeRange represents a time range within a day.
type TimeRange struct {
	StartTime string `yaml:"start_time,omitempty"`
	EndTime   string `yaml:"end_time,omitempty"`
}

const apiAlertsPath = "/api/v1/alerts"

// GetAlertmanagerConfig retrieves the alertmanager configuration.
func (c *Client) GetAlertmanagerConfig(ctx context.Context, orgID string) (*AlertmanagerUserConfig, error) {
	headers := make(map[string]string)
	if orgID != "" {
		headers["X-Scope-OrgID"] = orgID
	}

	resp, err := c.sendRequest(ctx, "alertmanager", http.MethodGet, apiAlertsPath, "", headers)
	if err != nil {
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get alertmanager config: %w", err)
	}

	var config AlertmanagerUserConfig
	if err := yaml.Unmarshal([]byte(resp), &config); err != nil {
		return nil, fmt.Errorf("failed to parse alertmanager config: %w", err)
	}

	return &config, nil
}

// SetAlertmanagerConfig sets the alertmanager configuration.
func (c *Client) SetAlertmanagerConfig(ctx context.Context, orgID string, config *AlertmanagerUserConfig) error {
	headers := map[string]string{"Content-Type": "application/yaml"}
	if orgID != "" {
		headers["X-Scope-OrgID"] = orgID
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal alertmanager config: %w", err)
	}

	_, err = c.sendRequest(ctx, "alertmanager", http.MethodPost, apiAlertsPath, string(data), headers)
	if err != nil {
		return fmt.Errorf("failed to set alertmanager config: %w", err)
	}

	return nil
}

// DeleteAlertmanagerConfig deletes the alertmanager configuration.
func (c *Client) DeleteAlertmanagerConfig(ctx context.Context, orgID string) error {
	headers := make(map[string]string)
	if orgID != "" {
		headers["X-Scope-OrgID"] = orgID
	}

	_, err := c.sendRequest(ctx, "alertmanager", http.MethodDelete, apiAlertsPath, "", headers)
	if err != nil && !IsNotFound(err) {
		return fmt.Errorf("failed to delete alertmanager config: %w", err)
	}

	return nil
}

// RuleGroup represents a Prometheus/Mimir rule group.
type RuleGroup struct {
	Name          string   `yaml:"name"`
	Interval      string   `yaml:"interval,omitempty"`
	SourceTenants []string `yaml:"source_tenants,omitempty"`
	Rules         []Rule   `yaml:"rules"`
}

// Rule represents a single alerting or recording rule.
type Rule struct {
	Alert         string            `yaml:"alert,omitempty"`
	Record        string            `yaml:"record,omitempty"`
	Expr          string            `yaml:"expr"`
	For           string            `yaml:"for,omitempty"`
	KeepFiringFor string            `yaml:"keep_firing_for,omitempty"`
	Labels        map[string]string `yaml:"labels,omitempty"`
	Annotations   map[string]string `yaml:"annotations,omitempty"`
}

// GetRuleGroup retrieves a rule group by namespace and name.
func (c *Client) GetRuleGroup(ctx context.Context, namespace, name, orgID string) (*RuleGroup, error) {
	headers := make(map[string]string)
	if orgID != "" {
		headers["X-Scope-OrgID"] = orgID
	}

	path := fmt.Sprintf("/config/v1/rules/%s/%s", url.PathEscape(namespace), url.PathEscape(name))
	resp, err := c.sendRequest(ctx, "ruler", http.MethodGet, path, "", headers)
	if err != nil {
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get rule group: %w", err)
	}

	var rg RuleGroup
	if err := yaml.Unmarshal([]byte(resp), &rg); err != nil {
		return nil, fmt.Errorf("failed to parse rule group: %w", err)
	}

	return &rg, nil
}

// SetRuleGroup creates or updates a rule group.
func (c *Client) SetRuleGroup(ctx context.Context, namespace, orgID string, rg *RuleGroup) error {
	headers := map[string]string{"Content-Type": "application/yaml"}
	if orgID != "" {
		headers["X-Scope-OrgID"] = orgID
	}

	data, err := yaml.Marshal(rg)
	if err != nil {
		return fmt.Errorf("failed to marshal rule group: %w", err)
	}

	path := fmt.Sprintf("/config/v1/rules/%s", url.PathEscape(namespace))
	_, err = c.sendRequest(ctx, "ruler", http.MethodPost, path, string(data), headers)
	if err != nil {
		return fmt.Errorf("failed to set rule group: %w", err)
	}

	return nil
}

// DeleteRuleGroup deletes a rule group.
func (c *Client) DeleteRuleGroup(ctx context.Context, namespace, name, orgID string) error {
	headers := make(map[string]string)
	if orgID != "" {
		headers["X-Scope-OrgID"] = orgID
	}

	path := fmt.Sprintf("/config/v1/rules/%s/%s", url.PathEscape(namespace), url.PathEscape(name))
	_, err := c.sendRequest(ctx, "ruler", http.MethodDelete, path, "", headers)
	if err != nil && !IsNotFound(err) {
		return fmt.Errorf("failed to delete rule group: %w", err)
	}

	return nil
}
