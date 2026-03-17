package config

import (
	"context"
	"fmt"

	"github.com/crossplane/upjet/v2/pkg/config"
)

// ExternalNameConfigs contains all external name configurations for this
// provider.
var ExternalNameConfigs = map[string]config.ExternalName{
	// Alertmanager configuration - uses org_id as the identifier
	// The Terraform provider returns the provider-level default org_id as the
	// resource ID, but we need to use the resource-level org_id parameter instead.
	// Users specify org_id in spec.forProvider.orgId, and we extract it from
	// tfstate to populate the external-name annotation correctly.
	"mimir_alertmanager_config": config.ExternalName{
		SetIdentifierArgumentFn: config.NopSetIdentifierArgument,
		GetExternalNameFn: func(tfstate map[string]any) (string, error) {
			// Extract from org_id field instead of id, because the Terraform provider
			// incorrectly returns the provider-level default org_id in the id field
			if orgID, ok := tfstate["org_id"].(string); ok && orgID != "" {
				return orgID, nil
			}
			// Fallback to id if org_id is not present in state
			if id, ok := tfstate["id"].(string); ok && id != "" {
				return id, nil
			}
			return "", fmt.Errorf("org_id and id are both empty in tfstate")
		},
		GetIDFn: func(_ context.Context, externalName string, _ map[string]any, _ map[string]any) (string, error) {
			// The external name is the org_id, which is also used as the Terraform ID
			return externalName, nil
		},
	},

	// Rule groups - use a composite identifier: namespace/name
	"mimir_rule_group_alerting":  config.TemplatedStringAsIdentifier("name", "{{ .parameters.namespace }}/{{ .external_name }}"),
	"mimir_rule_group_recording": config.TemplatedStringAsIdentifier("name", "{{ .parameters.namespace }}/{{ .external_name }}"),

	// Rules resource - manages multiple rule groups from a file
	// Same issue as alertmanager config - uses org_id as the identifier
	// Users specify org_id in spec.forProvider.orgId, and we extract it from
	// tfstate to populate the external-name annotation correctly.
	"mimir_rules": config.ExternalName{
		SetIdentifierArgumentFn: config.NopSetIdentifierArgument,
		GetExternalNameFn: func(tfstate map[string]any) (string, error) {
			// Extract from org_id field instead of id, because the Terraform provider
			// incorrectly returns the provider-level default org_id in the id field
			if orgID, ok := tfstate["org_id"].(string); ok && orgID != "" {
				return orgID, nil
			}
			// Fallback to id if org_id is not present in state
			if id, ok := tfstate["id"].(string); ok && id != "" {
				return id, nil
			}
			return "", fmt.Errorf("org_id and id are both empty in tfstate")
		},
		GetIDFn: func(_ context.Context, externalName string, _ map[string]any, _ map[string]any) (string, error) {
			// The external name is the org_id, which is also used as the Terraform ID
			return externalName, nil
		},
	},
}

// ExternalNameConfigurations applies all external name configs listed in the
// table ExternalNameConfigs and sets the version of those resources to v1beta1
// assuming they will be tested.
func ExternalNameConfigurations() config.ResourceOption {
	return func(r *config.Resource) {
		if e, ok := ExternalNameConfigs[r.Name]; ok {
			r.ExternalName = e
		}
	}
}

// ExternalNameConfigured returns the list of all resources whose external name
// is configured manually.
func ExternalNameConfigured() []string {
	l := make([]string, len(ExternalNameConfigs))
	i := 0
	for name := range ExternalNameConfigs {
		// $ is added to match the exact string since the format is regex.
		l[i] = name + "$"
		i++
	}
	return l
}
