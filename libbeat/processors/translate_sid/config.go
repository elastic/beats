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

package translate_sid

import "github.com/pkg/errors"

type config struct {
	Field             string `config:"field"  validate:"required"`
	AccountNameTarget string `config:"account_name_target"`
	AccountTypeTarget string `config:"account_type_target"`
	DomainTarget      string `config:"domain_target"`
	IgnoreMissing     bool   `config:"ignore_missing"`
	IgnoreFailure     bool   `config:"ignore_failure"`
}

func (c *config) Validate() error {
	if c.AccountNameTarget == "" && c.AccountTypeTarget == "" && c.DomainTarget == "" {
		return errors.New("at least one target field must be configured " +
			"(set account_name_target, account_type_target, and/or domain_target)")
	}
	return nil
}

func defaultConfig() config {
	return config{}
}
