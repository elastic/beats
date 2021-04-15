// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"strings"
	"sync"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/sorted"
)

// PackageController holds the information about used packages.
type PackageController struct {
	lock     sync.Mutex
	packages []string
}

func newPackageController() *PackageController {
	return &PackageController{
		packages: make([]string, 0),
	}
}

// Packages retrieves list of packages used by agent.
func (pc *PackageController) Packages() []string {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	return pc.packages
}

// ReloadPackages reloads list of packages from configuration in form of AST.
func (pc *PackageController) ReloadPackages(packages []string) error {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	sortedPackages := sorted.NewSet()
	for _, p := range packages {
		sortedPackages.Add(p, struct{}{})
	}

	pc.packages = sortedPackages.Keys()
	return nil

}

// Reload reloads list of packages from configuration in form of ucfg config.
func (pc *PackageController) Reload(rawConfig *config.Config) error {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	cfg := defaultMetaConfig()

	if err := rawConfig.Unpack(&cfg); err != nil {
		return err
	}

	pc.updatePackages(cfg)
	return nil
}

func (pc *PackageController) updatePackages(pp *metaConfig) {
	// deduplicate
	set := sorted.NewSet()

	for _, p := range pp.Inputs {
		metaName := strings.TrimSpace(p.MetaName)
		if metaName == "" {
			continue
		}

		set.Add(metaName, struct{}{})
	}

	pc.packages = set.Keys()
}

func defaultMetaConfig() *metaConfig {
	return &metaConfig{
		Inputs: make([]inputConfig, 0),
	}
}

type metaConfig struct {
	Inputs []inputConfig `config:"inputs"`
}

type inputConfig struct {
	MetaName string `config:"meta.package.name"`
}
