// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ipfix

import (
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/fields"
)

// Config contains the parquet reader config options.
type Config struct {
	// InternalNetworks defines the pre-configured networks treated as internal
	InternalNetworks []string `config:"internal_networks"`
	// CustomDefinitions
	CustomDefinitions []string `config:"custom_definitions"`
}

func (cfg *Config) Fields() fields.FieldDict {
	myFields := fields.FieldDict{}
	for _, yamlPath := range cfg.CustomDefinitions {
		f, err := decoder.LoadFieldDefinitionsFromFile(yamlPath)
		if err != nil {
			return nil
		}
		myFields.Merge(f)
	}

	return myFields
}
