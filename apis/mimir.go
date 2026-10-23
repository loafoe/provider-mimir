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

// Package apis contains Kubernetes API for the Mimir provider.
package apis

import (
	"k8s.io/apimachinery/pkg/runtime"

	alertmanagerv1 "github.com/loafoe/provider-mimir/apis/alertmanager/v1"
	malertmanagerv1 "github.com/loafoe/provider-mimir/apis/m/alertmanager/v1"
	mrulerv1 "github.com/loafoe/provider-mimir/apis/m/ruler/v1"
	mv1 "github.com/loafoe/provider-mimir/apis/m/v1"
	rulerv1 "github.com/loafoe/provider-mimir/apis/ruler/v1"
	v1 "github.com/loafoe/provider-mimir/apis/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	// Cluster-scoped resources (mimir.crossplane.io)
	AddToSchemes = append(AddToSchemes,
		v1.AddToScheme,
		alertmanagerv1.AddToScheme,
		rulerv1.AddToScheme,
	)
	// Namespaced resources (mimir.m.crossplane.io)
	AddToSchemes = append(AddToSchemes,
		mv1.AddToScheme,
		malertmanagerv1.AddToScheme,
		mrulerv1.AddToScheme,
	)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
