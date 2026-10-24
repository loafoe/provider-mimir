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
	"context"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Supported authentication types.
const (
	AuthTypeBasic = "basic"
	AuthTypeToken = "token"
)

// BasicAuthRefs references the secrets holding basic authentication credentials.
type BasicAuthRefs struct {
	UsernameSecretRef xpv1.SecretKeySelector
	PasswordSecretRef xpv1.SecretKeySelector
}

// TokenAuthRefs references the secret holding a bearer token.
type TokenAuthRefs struct {
	TokenSecretRef xpv1.SecretKeySelector
}

// TLSRefs references the secrets holding TLS material.
type TLSRefs struct {
	Insecure      bool
	CASecretRef   *xpv1.SecretKeySelector
	CertSecretRef *xpv1.SecretKeySelector
	KeySecretRef  *xpv1.SecretKeySelector
}

// CredentialsRefs describes how a Mimir client should authenticate.
type CredentialsRefs struct {
	AuthType  string
	BasicAuth *BasicAuthRefs
	TokenAuth *TokenAuthRefs
}

// BuildConfig resolves credential and TLS secrets and returns a complete Mimir
// client Config. The supplied cfg provides the non-secret settings.
func BuildConfig(ctx context.Context, kube client.Client, namespace string, cfg Config, creds CredentialsRefs, tlsRefs *TLSRefs) (Config, error) {
	if err := applyAuth(ctx, kube, namespace, &cfg, creds); err != nil {
		return cfg, err
	}
	if err := applyTLS(ctx, kube, namespace, &cfg, tlsRefs); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func applyAuth(ctx context.Context, kube client.Client, namespace string, cfg *Config, creds CredentialsRefs) error {
	switch creds.AuthType {
	case AuthTypeBasic:
		if creds.BasicAuth == nil {
			return errors.New("basicAuth is required when authType is basic")
		}
		username, err := getSecretValue(ctx, kube, namespace, creds.BasicAuth.UsernameSecretRef)
		if err != nil {
			return errors.Wrap(err, "cannot get username from secret")
		}
		password, err := getSecretValue(ctx, kube, namespace, creds.BasicAuth.PasswordSecretRef)
		if err != nil {
			return errors.Wrap(err, "cannot get password from secret")
		}
		cfg.Username = username
		cfg.Password = password
	case AuthTypeToken:
		if creds.TokenAuth == nil {
			return errors.New("tokenAuth is required when authType is token")
		}
		token, err := getSecretValue(ctx, kube, namespace, creds.TokenAuth.TokenSecretRef)
		if err != nil {
			return errors.Wrap(err, "cannot get token from secret")
		}
		cfg.Token = token
	default:
		return errors.Errorf("unsupported auth type: %s", creds.AuthType)
	}
	return nil
}

func applyTLS(ctx context.Context, kube client.Client, namespace string, cfg *Config, tlsRefs *TLSRefs) error {
	if tlsRefs == nil {
		return nil
	}
	cfg.Insecure = tlsRefs.Insecure
	if tlsRefs.CASecretRef != nil {
		ca, err := getSecretValue(ctx, kube, namespace, *tlsRefs.CASecretRef)
		if err != nil {
			return errors.Wrap(err, "cannot get CA from secret")
		}
		cfg.CA = ca
	}
	if tlsRefs.CertSecretRef != nil {
		cert, err := getSecretValue(ctx, kube, namespace, *tlsRefs.CertSecretRef)
		if err != nil {
			return errors.Wrap(err, "cannot get cert from secret")
		}
		cfg.Cert = cert
	}
	if tlsRefs.KeySecretRef != nil {
		key, err := getSecretValue(ctx, kube, namespace, *tlsRefs.KeySecretRef)
		if err != nil {
			return errors.Wrap(err, "cannot get key from secret")
		}
		cfg.Key = key
	}
	return nil
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
