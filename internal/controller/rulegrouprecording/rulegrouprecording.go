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

package rulegrouprecording

import (
	"context"
	"fmt"
	"strings"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rulerv1alpha1 "github.com/loafoe/provider-mimir/apis/ruler/v1alpha1"
	apisv1alpha1 "github.com/loafoe/provider-mimir/apis/v1alpha1"
	"github.com/loafoe/provider-mimir/internal/clients/mimir"
)

const (
	errNotRuleGroupRecording = "managed resource is not a RuleGroupRecording custom resource"
	errTrackPCUsage          = "cannot track ProviderConfig usage"
	errGetPC                 = "cannot get ProviderConfig"
	errNewClient             = "cannot create Mimir client"
	errInvalidExternalName   = "invalid external name format, expected namespace/name"
)

func formatExternalName(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func parseExternalName(externalName string) (string, string, error) {
	parts := strings.SplitN(externalName, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errors.New(errInvalidExternalName)
	}
	return parts[0], parts[1], nil
}

// Setup adds a controller that reconciles RuleGroupRecording managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(rulerv1alpha1.RuleGroupRecordingGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}
	if o.Features.Enabled(feature.EnableAlphaChangeLogs) {
		opts = append(opts, managed.WithChangeLogger(o.ChangeLogOptions.ChangeLogger))
	}
	if o.MetricOptions != nil {
		opts = append(opts, managed.WithMetricRecorder(o.MetricOptions.MRMetrics))
	}
	if o.MetricOptions != nil && o.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &rulerv1alpha1.RuleGroupRecordingList{}, o.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder")
		}
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(rulerv1alpha1.RuleGroupRecordingGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&rulerv1alpha1.RuleGroupRecording{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*rulerv1alpha1.RuleGroupRecording)
	if !ok {
		return nil, errors.New(errNotRuleGroupRecording)
	}

	if err := c.usage.Track(ctx, cr); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	m := mg.(resource.ModernManaged)
	ref := m.GetProviderConfigReference()

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: m.GetNamespace()}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cfg, err := buildClientConfig(ctx, c.kube, m.GetNamespace(), pc)
	if err != nil {
		return nil, err
	}

	mimirClient, err := mimir.NewClient(cfg)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: mimirClient, defaultOrgID: pc.Spec.OrgID}, nil
}

func buildClientConfig(ctx context.Context, kube client.Client, namespace string, pc *apisv1alpha1.ProviderConfig) (mimir.Config, error) {
	cfg := mimir.Config{
		URI:             pc.Spec.URI,
		RulerURI:        pc.Spec.RulerURI,
		AlertmanagerURI: pc.Spec.AlertmanagerURI,
		OrgID:           pc.Spec.OrgID,
		Headers:         pc.Spec.Headers,
	}

	switch pc.Spec.Credentials.AuthType {
	case apisv1alpha1.AuthTypeBasic:
		if pc.Spec.Credentials.BasicAuth == nil {
			return cfg, errors.New("basicAuth is required when authType is basic")
		}
		username, err := getSecretValue(ctx, kube, namespace, pc.Spec.Credentials.BasicAuth.UsernameSecretRef)
		if err != nil {
			return cfg, errors.Wrap(err, "cannot get username from secret")
		}
		password, err := getSecretValue(ctx, kube, namespace, pc.Spec.Credentials.BasicAuth.PasswordSecretRef)
		if err != nil {
			return cfg, errors.Wrap(err, "cannot get password from secret")
		}
		cfg.Username = username
		cfg.Password = password
	case apisv1alpha1.AuthTypeToken:
		if pc.Spec.Credentials.TokenAuth == nil {
			return cfg, errors.New("tokenAuth is required when authType is token")
		}
		token, err := getSecretValue(ctx, kube, namespace, pc.Spec.Credentials.TokenAuth.TokenSecretRef)
		if err != nil {
			return cfg, errors.Wrap(err, "cannot get token from secret")
		}
		cfg.Token = token
	default:
		return cfg, errors.Errorf("unsupported auth type: %s", pc.Spec.Credentials.AuthType)
	}

	if pc.Spec.TLS != nil {
		cfg.Insecure = pc.Spec.TLS.Insecure
		if pc.Spec.TLS.CASecretRef != nil {
			ca, err := getSecretValue(ctx, kube, namespace, *pc.Spec.TLS.CASecretRef)
			if err != nil {
				return cfg, errors.Wrap(err, "cannot get CA from secret")
			}
			cfg.CA = ca
		}
		if pc.Spec.TLS.CertSecretRef != nil {
			cert, err := getSecretValue(ctx, kube, namespace, *pc.Spec.TLS.CertSecretRef)
			if err != nil {
				return cfg, errors.Wrap(err, "cannot get cert from secret")
			}
			cfg.Cert = cert
		}
		if pc.Spec.TLS.KeySecretRef != nil {
			key, err := getSecretValue(ctx, kube, namespace, *pc.Spec.TLS.KeySecretRef)
			if err != nil {
				return cfg, errors.Wrap(err, "cannot get key from secret")
			}
			cfg.Key = key
		}
	}

	return cfg, nil
}

func getSecretValue(ctx context.Context, kube client.Client, namespace string, ref xpv1.SecretKeySelector) (string, error) {
	secretRef := ref
	if secretRef.Namespace == "" {
		secretRef.Namespace = namespace
	}
	data, err := resource.CommonCredentialExtractor(ctx, xpv1.CredentialsSourceSecret, kube, xpv1.CommonCredentialSelectors{SecretRef: &secretRef})
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type external struct {
	client       *mimir.Client
	defaultOrgID string
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*rulerv1alpha1.RuleGroupRecording)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRuleGroupRecording)
	}

	fp := cr.Spec.ForProvider
	orgID := e.getOrgID(cr)
	namespace := "default"
	if fp.Namespace != nil {
		namespace = *fp.Namespace
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	_, name, err := parseExternalName(externalName)
	if err != nil {
		name = fp.Name
	}

	rg, err := e.client.GetRuleGroup(ctx, namespace, name, orgID)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get rule group")
	}
	if rg == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider.OrgID = &orgID
	cr.Status.AtProvider.Namespace = &namespace
	cr.Status.AtProvider.Name = &rg.Name
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: e.isUpToDate(cr, rg),
	}, nil
}

func (e *external) isUpToDate(cr *rulerv1alpha1.RuleGroupRecording, rg *mimir.RuleGroup) bool {
	fp := cr.Spec.ForProvider

	if fp.Name != rg.Name {
		return false
	}

	if fp.Interval != nil && *fp.Interval != rg.Interval {
		return false
	}

	if len(fp.Rules) != len(rg.Rules) {
		return false
	}

	return true
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*rulerv1alpha1.RuleGroupRecording)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRuleGroupRecording)
	}

	cr.Status.SetConditions(xpv1.Creating())

	fp := cr.Spec.ForProvider
	orgID := e.getOrgID(cr)
	namespace := "default"
	if fp.Namespace != nil {
		namespace = *fp.Namespace
	}

	rg := e.buildRuleGroup(cr)

	if err := e.client.SetRuleGroup(ctx, namespace, orgID, rg); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create rule group")
	}

	meta.SetExternalName(cr, formatExternalName(namespace, fp.Name))
	cr.Status.AtProvider.OrgID = &orgID
	cr.Status.AtProvider.Namespace = &namespace
	cr.Status.AtProvider.Name = &fp.Name

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*rulerv1alpha1.RuleGroupRecording)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRuleGroupRecording)
	}

	fp := cr.Spec.ForProvider
	orgID := e.getOrgID(cr)
	namespace := "default"
	if fp.Namespace != nil {
		namespace = *fp.Namespace
	}

	rg := e.buildRuleGroup(cr)

	if err := e.client.SetRuleGroup(ctx, namespace, orgID, rg); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update rule group")
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*rulerv1alpha1.RuleGroupRecording)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRuleGroupRecording)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	fp := cr.Spec.ForProvider
	orgID := e.getOrgID(cr)
	namespace := "default"
	if fp.Namespace != nil {
		namespace = *fp.Namespace
	}

	if err := e.client.DeleteRuleGroup(ctx, namespace, fp.Name, orgID); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete rule group")
	}

	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(ctx context.Context) error {
	return nil
}

func (e *external) getOrgID(cr *rulerv1alpha1.RuleGroupRecording) string {
	if cr.Spec.ForProvider.OrgID != nil {
		return *cr.Spec.ForProvider.OrgID
	}
	return e.defaultOrgID
}

func (e *external) buildRuleGroup(cr *rulerv1alpha1.RuleGroupRecording) *mimir.RuleGroup {
	fp := cr.Spec.ForProvider

	rg := &mimir.RuleGroup{
		Name:          fp.Name,
		SourceTenants: fp.SourceTenants,
	}

	if fp.Interval != nil {
		rg.Interval = *fp.Interval
	}

	for _, r := range fp.Rules {
		rule := mimir.Rule{
			Record: r.Record,
			Expr:   r.Expr,
			Labels: r.Labels,
		}
		rg.Rules = append(rg.Rules, rule)
	}

	return rg
}
