// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"strings"
	"sync"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/sorted"
)

type packageController struct {
	lock     sync.Mutex
	packages []string
}

func newPackageController() *packageController {
	return &packageController{
		packages: make([]string, 0),
	}
}

func (pc *packageController) Packages() []string {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	return pc.packages
}

func (pc *packageController) ReloadAST(rootAst *transpiler.AST) error {
	mm, err := rootAst.Map()
	if err != nil {
		return err
	}

	cfg, err := config.NewConfigFrom(mm)
	if err != nil {
		return err
	}

	return pc.Reload(cfg)

}

func (pc *packageController) Reload(rawConfig *config.Config) error {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	cfg := defaultMetaConfig()

	if err := rawConfig.Unpack(&cfg); err != nil {
		return err
	}

	pc.updatePackages(cfg)
	return nil
}

func (pc *packageController) updatePackages(pp *metaConfig) {
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
