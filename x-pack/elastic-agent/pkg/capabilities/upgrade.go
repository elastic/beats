// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/eql"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

const (
	versionKey   = "version"
	sourceURIKey = "source_uri"
)

type upgradeCapability struct {
	Type string `json:"rule" yaml:"rule"`
	// UpgradeEql is eql expression defining upgrade
	UpgradeEqlDefinition string `json:"upgrade" yaml:"upgrade"`

	upgradeEql *eql.Expression
}

func (c *upgradeCapability) Rule() string {
	return c.Type
}

// Apply supports upgrade action or fleetapi upgrade action object.
func (c *upgradeCapability) Apply(in interface{}) (bool, interface{}) {
	// if eql is not parsed or defined skip
	if c.upgradeEql == nil {
		return false, in
	}

	upgradeMap := upgradeObject(in)
	if upgradeMap == nil {
		// TODO: log warning
		// not an upgrade we don't alter origin
		return false, in
	}

	// create VarStore out of map
	varStore, err := transpiler.NewAST(upgradeMap)
	if err != nil {
		// TODO: log error
		return false, in
	}

	isSupported, err := c.upgradeEql.Eval(varStore)
	if err != nil {
		// TODO: log error
		return false, in
	}

	// if deny switch the logic
	if c.Type == denyKey {
		isSupported = !isSupported
	}

	return !isSupported, in
}

// NewUpgradeCapability creates capability filter for upgrade.
// Available variables:
// - version
// - source_uri
func NewUpgradeCapability(r ruler) (Capability, error) {
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
	return cap, nil
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
