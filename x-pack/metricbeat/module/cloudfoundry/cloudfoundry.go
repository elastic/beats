// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	cfcommon "github.com/elastic/beats/v7/x-pack/libbeat/common/cloudfoundry"
)

// ModuleName is the name of this module.
const ModuleName = "cloudfoundry"

func init() {
	if err := mb.Registry.AddModule(ModuleName, newModule); err != nil {
		panic(err)
	}
}

type Module interface {
	RunCounterReporter(mb.PushReporterV2)
	RunContainerReporter(mb.PushReporterV2)
	RunValueReporter(mb.PushReporterV2)
}

func newModule(base mb.BaseModule) (mb.Module, error) {
	var cfg cfcommon.Config
	if err := base.UnpackConfig(&cfg); err != nil {
		return nil, err
	}

	log := logp.NewLogger("cloudfoundry")
	hub := cfcommon.NewHub(&cfg, "metricbeat", log)

	switch cfg.Version {
	case cfcommon.ConsumerVersionV1:
		return newModuleV1(base, hub, log)
	case cfcommon.ConsumerVersionV2:
		return newModuleV2(base, hub, log)
	default:
		return nil, fmt.Errorf("not supported consumer version: %s", cfg.Version)
	}
}
