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

package alertmanagerconfig

import (
	"context"

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
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	alertmanagerv1alpha1 "github.com/loafoe/provider-mimir/apis/alertmanager/v1alpha1"
	apisv1alpha1 "github.com/loafoe/provider-mimir/apis/v1alpha1"
	"github.com/loafoe/provider-mimir/internal/clients/mimir"
)

const (
	errNotConfig       = "managed resource is not an AlertmanagerConfig custom resource"
	errTrackPCUsage    = "cannot track ProviderConfig usage"
	errGetPC           = "cannot get ProviderConfig"
	errNewClient       = "cannot create Mimir client"
	errGetSecret       = "cannot get secret"
)

// Setup adds a controller that reconciles AlertmanagerConfig managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(alertmanagerv1alpha1.ConfigGroupKind)

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
			mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &alertmanagerv1alpha1.ConfigList{}, o.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder")
		}
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(alertmanagerv1alpha1.ConfigGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&alertmanagerv1alpha1.Config{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*alertmanagerv1alpha1.Config)
	if !ok {
		return nil, errors.New(errNotConfig)
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

	cfg, err := c.buildClientConfig(ctx, m.GetNamespace(), pc)
	if err != nil {
		return nil, err
	}

	mimirClient, err := mimir.NewClient(cfg)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: mimirClient, defaultOrgID: pc.Spec.OrgID}, nil
}

func (c *connector) buildClientConfig(ctx context.Context, namespace string, pc *apisv1alpha1.ProviderConfig) (mimir.Config, error) {
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
		username, err := c.getSecretValue(ctx, namespace, pc.Spec.Credentials.BasicAuth.UsernameSecretRef)
		if err != nil {
			return cfg, errors.Wrap(err, "cannot get username from secret")
		}
		password, err := c.getSecretValue(ctx, namespace, pc.Spec.Credentials.BasicAuth.PasswordSecretRef)
		if err != nil {
			return cfg, errors.Wrap(err, "cannot get password from secret")
		}
		cfg.Username = username
		cfg.Password = password
	case apisv1alpha1.AuthTypeToken:
		if pc.Spec.Credentials.TokenAuth == nil {
			return cfg, errors.New("tokenAuth is required when authType is token")
		}
		token, err := c.getSecretValue(ctx, namespace, pc.Spec.Credentials.TokenAuth.TokenSecretRef)
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
			ca, err := c.getSecretValue(ctx, namespace, *pc.Spec.TLS.CASecretRef)
			if err != nil {
				return cfg, errors.Wrap(err, "cannot get CA from secret")
			}
			cfg.CA = ca
		}
		if pc.Spec.TLS.CertSecretRef != nil {
			cert, err := c.getSecretValue(ctx, namespace, *pc.Spec.TLS.CertSecretRef)
			if err != nil {
				return cfg, errors.Wrap(err, "cannot get cert from secret")
			}
			cfg.Cert = cert
		}
		if pc.Spec.TLS.KeySecretRef != nil {
			key, err := c.getSecretValue(ctx, namespace, *pc.Spec.TLS.KeySecretRef)
			if err != nil {
				return cfg, errors.Wrap(err, "cannot get key from secret")
			}
			cfg.Key = key
		}
	}

	return cfg, nil
}

func (c *connector) getSecretValue(ctx context.Context, namespace string, ref xpv1.SecretKeySelector) (string, error) {
	secretRef := ref
	if secretRef.Namespace == "" {
		secretRef.Namespace = namespace
	}
	data, err := resource.CommonCredentialExtractor(ctx, xpv1.CredentialsSourceSecret, c.kube, xpv1.CommonCredentialSelectors{SecretRef: &secretRef})
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
	cr, ok := mg.(*alertmanagerv1alpha1.Config)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotConfig)
	}

	orgID := e.getOrgID(cr)
	externalName := meta.GetExternalName(cr)

	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	config, err := e.client.GetAlertmanagerConfig(ctx, orgID)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get alertmanager config")
	}
	if config == nil || config.AlertmanagerConfig == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider.OrgID = &orgID
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true, // TODO: implement proper diff checking
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*alertmanagerv1alpha1.Config)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotConfig)
	}

	cr.Status.SetConditions(xpv1.Creating())

	orgID := e.getOrgID(cr)
	config := e.buildAlertmanagerConfig(cr)

	if err := e.client.SetAlertmanagerConfig(ctx, orgID, config); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create alertmanager config")
	}

	meta.SetExternalName(cr, orgID)
	cr.Status.AtProvider.OrgID = &orgID

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*alertmanagerv1alpha1.Config)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotConfig)
	}

	orgID := e.getOrgID(cr)
	config := e.buildAlertmanagerConfig(cr)

	if err := e.client.SetAlertmanagerConfig(ctx, orgID, config); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update alertmanager config")
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*alertmanagerv1alpha1.Config)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotConfig)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	orgID := e.getOrgID(cr)

	if err := e.client.DeleteAlertmanagerConfig(ctx, orgID); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete alertmanager config")
	}

	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(ctx context.Context) error {
	return nil
}

func (e *external) getOrgID(cr *alertmanagerv1alpha1.Config) string {
	if cr.Spec.ForProvider.OrgID != nil {
		return *cr.Spec.ForProvider.OrgID
	}
	return e.defaultOrgID
}

func (e *external) buildAlertmanagerConfig(cr *alertmanagerv1alpha1.Config) *mimir.AlertmanagerUserConfig {
	fp := cr.Spec.ForProvider

	alertConfig := &mimir.AlertmanagerConfig{
		Templates: fp.Templates,
	}

	if fp.Global != nil {
		alertConfig.Global = &mimir.GlobalConfig{
			ResolveTimeout: ptrToStr(fp.Global.ResolveTimeout),
			SMTPSmarthost:  ptrToStr(fp.Global.SMTPSmarthost),
			SMTPFrom:       ptrToStr(fp.Global.SMTPFrom),
			SMTPAuthUsername: ptrToStr(fp.Global.SMTPAuthUsername),
			SMTPAuthIdentity: ptrToStr(fp.Global.SMTPAuthIdentity),
			SMTPRequireTLS: fp.Global.SMTPRequireTLS,
			PagerdutyURL:  ptrToStr(fp.Global.PagerdutyURL),
			OpsGenieAPIURL: ptrToStr(fp.Global.OpsGenieAPIURL),
		}
	}

	if fp.Route != nil {
		alertConfig.Route = e.buildRoute(fp.Route)
	}

	for _, r := range fp.Receivers {
		receiver := mimir.Receiver{Name: r.Name}
		for _, ec := range r.EmailConfigs {
			receiver.EmailConfigs = append(receiver.EmailConfigs, mimir.EmailConfig{
				To:           ptrToStr(ec.To),
				From:         ptrToStr(ec.From),
				Smarthost:    ptrToStr(ec.Smarthost),
				AuthUsername: ptrToStr(ec.AuthUsername),
				HTML:         ptrToStr(ec.HTML),
				Text:         ptrToStr(ec.Text),
				RequireTLS:   ec.RequireTLS,
				SendResolved: ptrToBool(ec.SendResolved),
			})
		}
		for _, sc := range r.SlackConfigs {
			receiver.SlackConfigs = append(receiver.SlackConfigs, mimir.SlackConfig{
				Channel:      ptrToStr(sc.Channel),
				Username:     ptrToStr(sc.Username),
				IconEmoji:    ptrToStr(sc.IconEmoji),
				IconURL:      ptrToStr(sc.IconURL),
				Title:        ptrToStr(sc.Title),
				Text:         ptrToStr(sc.Text),
				SendResolved: ptrToBool(sc.SendResolved),
			})
		}
		for _, wc := range r.WebhookConfigs {
			receiver.WebhookConfigs = append(receiver.WebhookConfigs, mimir.WebhookConfig{
				URL:          ptrToStr(wc.URL),
				SendResolved: ptrToBool(wc.SendResolved),
			})
		}
		alertConfig.Receivers = append(alertConfig.Receivers, receiver)
	}

	for _, ir := range fp.InhibitRules {
		alertConfig.InhibitRules = append(alertConfig.InhibitRules, mimir.InhibitRule{
			SourceMatchers: ir.SourceMatchers,
			TargetMatchers: ir.TargetMatchers,
			Equal:          ir.Equal,
		})
	}

	for _, ti := range fp.TimeIntervals {
		mti := mimir.MuteTimeInterval{Name: ti.Name}
		for _, interval := range ti.TimeIntervals {
			t := mimir.TimeInterval{
				Weekdays:    interval.Weekdays,
				DaysOfMonth: interval.DaysOfMonth,
				Months:      interval.Months,
				Years:       interval.Years,
			}
			for _, tr := range interval.Times {
				t.Times = append(t.Times, mimir.TimeRange{
					StartTime: ptrToStr(tr.StartTime),
					EndTime:   ptrToStr(tr.EndTime),
				})
			}
			mti.TimeIntervals = append(mti.TimeIntervals, t)
		}
		alertConfig.MuteTimeIntervals = append(alertConfig.MuteTimeIntervals, mti)
	}

	return &mimir.AlertmanagerUserConfig{
		TemplateFiles:      fp.TemplatesFiles,
		AlertmanagerConfig: mustMarshalYAML(alertConfig),
	}
}

func (e *external) buildRoute(rc *alertmanagerv1alpha1.RouteConfig) *mimir.Route {
	if rc == nil {
		return nil
	}
	route := &mimir.Route{
		Receiver:            ptrToStr(rc.Receiver),
		GroupBy:             rc.GroupBy,
		GroupWait:           ptrToStr(rc.GroupWait),
		GroupInterval:       ptrToStr(rc.GroupInterval),
		RepeatInterval:      ptrToStr(rc.RepeatInterval),
		Continue:            ptrToBool(rc.Continue),
		Matchers:            rc.Matchers,
		MuteTimeIntervals:   rc.MuteTimeIntervals,
		ActiveTimeIntervals: rc.ActiveTimeIntervals,
	}
	for _, child := range rc.ChildRoute {
		route.Routes = append(route.Routes, mimir.Route{
			Receiver:            ptrToStr(child.Receiver),
			GroupBy:             child.GroupBy,
			GroupWait:           ptrToStr(child.GroupWait),
			GroupInterval:       ptrToStr(child.GroupInterval),
			RepeatInterval:      ptrToStr(child.RepeatInterval),
			Continue:            ptrToBool(child.Continue),
			Matchers:            child.Matchers,
			MuteTimeIntervals:   child.MuteTimeIntervals,
			ActiveTimeIntervals: child.ActiveTimeIntervals,
		})
	}
	return route
}

func ptrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrToBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func mustMarshalYAML(v interface{}) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}
