// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/upjet/v2/pkg/controller"

	config "github.com/loafoe/provider-mimir/internal/controller/cluster/alertmanager/config"
	providerconfig "github.com/loafoe/provider-mimir/internal/controller/cluster/providerconfig"
	groupalerting "github.com/loafoe/provider-mimir/internal/controller/cluster/ruler/groupalerting"
	grouprecording "github.com/loafoe/provider-mimir/internal/controller/cluster/ruler/grouprecording"
	rules "github.com/loafoe/provider-mimir/internal/controller/cluster/ruler/rules"
)

// Setup creates all controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		config.Setup,
		providerconfig.Setup,
		groupalerting.Setup,
		grouprecording.Setup,
		rules.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupGated creates all controllers with the supplied logger and adds them to
// the supplied manager gated.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		config.SetupGated,
		providerconfig.SetupGated,
		groupalerting.SetupGated,
		grouprecording.SetupGated,
		rules.SetupGated,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
