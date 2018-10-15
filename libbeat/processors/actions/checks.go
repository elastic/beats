// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package actions

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

func configChecked(
	constr processors.Constructor,
	checks ...func(*common.Config) error,
) processors.Constructor {
	validator := checkAll(checks...)
	return func(cfg *common.Config) (processors.Processor, error) {
		err := validator(cfg)
		if err != nil {
			return nil, fmt.Errorf("%v in %v", err.Error(), cfg.Path())
		}

		return constr(cfg)
	}
}

func checkAll(checks ...func(*common.Config) error) func(*common.Config) error {
	return func(c *common.Config) error {
		for _, check := range checks {
			if err := check(c); err != nil {
				return err
			}
		}
		return nil
	}
}

func requireFields(fields ...string) func(*common.Config) error {
	return func(cfg *common.Config) error {
		for _, field := range fields {
			if !cfg.HasField(field) {
				return fmt.Errorf("missing %v option", field)
			}
		}
		return nil
	}
}

func allowedFields(fields ...string) func(*common.Config) error {
	return func(cfg *common.Config) error {
		for _, field := range cfg.GetFields() {
			found := false
			for _, allowed := range fields {
				if field == allowed {
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("unexpected %v option", field)
			}
		}
		return nil
	}
}
