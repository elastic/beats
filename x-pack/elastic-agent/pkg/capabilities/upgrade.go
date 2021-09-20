// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"

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
		return &multiUpgradeCapability{caps: []*upgradeCapability{}}, nil
	}

	caps := make([]*upgradeCapability, 0, len(rd.Capabilities))

	for _, r := range rd.Capabilities {
		c, err := newUpgradeCapability(log, r, reporter)
		if err != nil {
			return nil, err
		}

		if c != nil {
			caps = append(caps, c)
		}
	}

	return &multiUpgradeCapability{log: log, caps: caps}, nil
}

func newUpgradeCapability(log *logger.Logger, r ruler, reporter status.Reporter) (*upgradeCapability, error) {
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
func (c *upgradeCapability) Apply(upgradeMap map[string]interface{}) (map[string]interface{}, error) {
	// if eql is not parsed or defined skip
	if c.upgradeEql == nil {
		return upgradeMap, nil
	}

	// create VarStore out of map
	varStore, err := transpiler.NewAST(upgradeMap)
	if err != nil {
		c.log.Errorf("failed creating a varStore for capability '%s': %v", c.name(), err)
		return upgradeMap, nil
	}

	isSupported, err := c.upgradeEql.Eval(varStore)
	if err != nil {
		c.log.Errorf("failed evaluating eql formula for capability '%s': %v", c.name(), err)
		return upgradeMap, nil
	}

	// if deny switch the logic
	if c.Type == denyKey {
		isSupported = !isSupported
		msg := fmt.Sprintf("upgrade is blocked out due to capability restriction '%s'", c.name())
		c.log.Errorf(msg)
		c.reporter.Update(state.Degraded, msg, nil)
	}

	if !isSupported {
		return upgradeMap, ErrBlocked
	}

	return upgradeMap, nil
}

type multiUpgradeCapability struct {
	log  *logger.Logger
	caps []*upgradeCapability
}

func (c *multiUpgradeCapability) Apply(in interface{}) (interface{}, error) {
	upgradeMap := upgradeObject(in)
	if upgradeMap == nil {
		// not an upgrade we don't alter origin
		return in, nil
	}

	for _, cap := range c.caps {
		// upgrade does not modify incoming action
		_, err := cap.Apply(upgradeMap)
		if err != nil {
			return in, err
		}
	}

	return in, nil
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
