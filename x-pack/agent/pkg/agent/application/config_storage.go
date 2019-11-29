// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import "github.com/elastic/beats/agent/kibana"

type fleetConfig struct {
	AccessAPIKey string         `config:"access_api_key"`
	Kibana       *kibana.Config `config:"kibana"`
}

type store interface {
	Save(fleetConfig) error
}

// NullStore this is only use to split the work into multiples PRs.
// TODO(ph) make real implementation this is just to make test green and iterate.
type NullStore struct{}

// Save takes the fleetConfig and persist it, will return an errors on failure.
func (m *NullStore) Save(_ fleetConfig) error {
	return nil
}
