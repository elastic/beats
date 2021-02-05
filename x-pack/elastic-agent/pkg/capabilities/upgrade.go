// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/eql"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

const (
	versionKey   = "version"
	sourceURIKey = "source_uri"
)

// NewUpgradeCapability creates capability filter for upgrade.
// Available variables:
// - version
// - source_uri
func newUpgradesCapability(log *logger.Logger, rd *ruleDefinitions, reporter status.Reporter) (Capability, error) {
	if rd == nil {
		return &multiInputsCapability{log: log, caps: []Capability{}}, nil
	}

	caps := make([]Capability, 0, len(rd.Capabilities))

	for _, r := range rd.Capabilities {
		c, err := newUpgradeCapability(log, r, reporter)
		if err != nil {
			return nil, err
		}

		if c != nil {
			caps = append(caps, c)
		}
	}

	return &multiUpgradeCapability{caps: caps}, nil
}

func newUpgradeCapability(log *logger.Logger, r ruler, reporter status.Reporter) (Capability, error) {
	cap, ok := r.(*upgradeCapability)
	if !ok {
		return nil, nil
	}

	cap.Type = strings.ToLower(cap.Type)
	if cap.Type != allowKey && cap.Type != denyKey {
		return nil, fmt.Errorf("'%s' is not a valid type 'allow' and 'deny' are supported", cap.Type)
	}

	// if eql definition is not supported make a global rule
	if len(cap.UpgradeEqlDefinition) == 0 {
		cap.UpgradeEqlDefinition = "true"
	}

	eqlExp, err := eql.New(cap.UpgradeEqlDefinition)
	if err != nil {
		return nil, err
	}

	cap.upgradeEql = eqlExp
	cap.log = log
	cap.reporter = reporter
	return cap, nil
}

type upgradeCapability struct {
	log      *logger.Logger
	reporter status.Reporter
	Name     string `json:"name,omitempty" yaml:"name,omitempty"`
	Type     string `json:"rule" yaml:"rule"`
	// UpgradeEql is eql expression defining upgrade
	UpgradeEqlDefinition string `json:"upgrade" yaml:"upgrade"`

	upgradeEql *eql.Expression
}

func (c *upgradeCapability) Rule() string {
	return c.Type
}

func (c *upgradeCapability) name() string {
	if c.Name != "" {
		return c.Name
	}

	t := "A"
	if c.Type == denyKey {
		t = "D"
	}

	// e.g UA(*) or UD(7.*.*)
	c.Name = fmt.Sprintf("U%s(%s)", t, c.UpgradeEqlDefinition)
	return c.Name
}

// Apply supports upgrade action or fleetapi upgrade action object.
func (c *upgradeCapability) Apply(in interface{}) (bool, interface{}) {
	// if eql is not parsed or defined skip
	if c.upgradeEql == nil {
		return false, in
	}

	upgradeMap := upgradeObject(in)
	if upgradeMap == nil {
		c.log.Warnf("expecting map config object but got nil for capability 'multi-outputs': %v", c.name())
		// not an upgrade we don't alter origin
		return false, in
	}

	// create VarStore out of map
	varStore, err := transpiler.NewAST(upgradeMap)
	if err != nil {
		c.log.Errorf("failed creating a varStore for capability '%s': %v", c.name(), err)
		return false, in
	}

	isSupported, err := c.upgradeEql.Eval(varStore)
	if err != nil {
		c.log.Errorf("failed evaluating eql formula for capability '%s': %v", c.name(), err)
		return false, in
	}

	// if deny switch the logic
	if c.Type == denyKey {
		isSupported = !isSupported
		c.log.Errorf("upgrade is blocked out due to capability restriction '%s'", c.name())
		c.reporter.Update(status.Degraded)
	}

	return !isSupported, in
}

type multiUpgradeCapability struct {
	caps []Capability
}

func (c *multiUpgradeCapability) Apply(in interface{}) (bool, interface{}) {
	for _, cap := range c.caps {
		// upgrade does not modify incoming action
		blocking, _ := cap.Apply(in)
		if blocking {
			return blocking, in
		}
	}

	return false, in
}

func upgradeObject(a interface{}) map[string]interface{} {
	resultMap := make(map[string]interface{})
	if ua, ok := a.(upgradeAction); ok {
		resultMap[versionKey] = ua.Version()
		resultMap[sourceURIKey] = ua.SourceURI()
		return resultMap
	}

	if ua, ok := a.(*fleetapi.ActionUpgrade); ok {
		resultMap[versionKey] = ua.Version
		resultMap[sourceURIKey] = ua.SourceURI
		return resultMap
	}

	if ua, ok := a.(fleetapi.ActionUpgrade); ok {
		resultMap[versionKey] = ua.Version
		resultMap[sourceURIKey] = ua.SourceURI
		return resultMap
	}

	return nil
}

type upgradeAction interface {
	// Version to upgrade to.
	Version() string
	// SourceURI for download.
	SourceURI() string
}
