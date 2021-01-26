// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import (
	"encoding/json"
	"fmt"
)

type ruler interface {
	Rule() string
}

type ruleDefinitions []ruler

func (r *ruleDefinitions) UnmarshalJSON(p []byte) (resErr error) {
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
