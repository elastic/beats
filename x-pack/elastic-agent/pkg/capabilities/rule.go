// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v2"
)

const (
	allowKey     = "allow"
	denyKey      = "deny"
	conditionKey = "__condition__"
)

type ruler interface {
	Rule() string
}

type capabilitiesList []ruler

type ruleDefinitions struct {
	Version      string           `yaml:"version" json:"version"`
	Capabilities capabilitiesList `yaml:"capabilities" json:"capabilities"`
}

func (r *capabilitiesList) UnmarshalJSON(p []byte) error {
	var tmpArray []json.RawMessage

	err := json.Unmarshal(p, &tmpArray)
	if err != nil {
		return err
	}

	for i, t := range tmpArray {
		mm := make(map[string]interface{})
		if err := json.Unmarshal(t, &mm); err != nil {
			return err
		}

		if _, found := mm["input"]; found {
			cap := &inputCapability{}
			if err := json.Unmarshal(t, &cap); err != nil {
				return err
			}
			(*r) = append((*r), cap)

		} else if _, found = mm["output"]; found {
			cap := &outputCapability{}
			if err := json.Unmarshal(t, &cap); err != nil {
				return err
			}
			(*r) = append((*r), cap)

		} else if _, found = mm["upgrade"]; found {
			cap := &upgradeCapability{}
			if err := json.Unmarshal(t, &cap); err != nil {
				return err
			}
			(*r) = append((*r), cap)
		} else {
			return fmt.Errorf("unexpected capability type for definition number '%d'", i)
		}
	}

	return nil
}

func (r *capabilitiesList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var tmpArray []map[string]interface{}

	err := unmarshal(&tmpArray)
	if err != nil {
		return err
	}

	for i, mm := range tmpArray {
		partialYaml, err := yaml.Marshal(mm)
		if err != nil {
			return err
		}
		if _, found := mm["input"]; found {
			cap := &inputCapability{}
			if err := yaml.Unmarshal(partialYaml, &cap); err != nil {
				return err
			}
			(*r) = append((*r), cap)

		} else if _, found = mm["output"]; found {
			cap := &outputCapability{}
			if err := yaml.Unmarshal(partialYaml, &cap); err != nil {
				return err
			}
			(*r) = append((*r), cap)

		} else if _, found = mm["upgrade"]; found {
			cap := &upgradeCapability{}
			if err := yaml.Unmarshal(partialYaml, &cap); err != nil {
				return err
			}
			(*r) = append((*r), cap)
		} else {
			return fmt.Errorf("unexpected capability type for definition number '%d'", i)
		}
	}

	return nil
}
