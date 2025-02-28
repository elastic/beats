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

//go:build !requirefips

package kerberos

import "fmt"

func (c *Config) Validate() error {
	switch c.AuthType {
	case authPassword:
		if c.Username == "" {
			return fmt.Errorf("password authentication is selected for Kerberos, but username is not configured")
		}
		if c.Password == "" {
			return fmt.Errorf("password authentication is selected for Kerberos, but password is not configured")
		}

	case authKeytab:
		if c.KeyTabPath == "" {
			return fmt.Errorf("keytab authentication is selected for Kerberos, but path to keytab is not configured")
		}
	default:
		return ErrInvalidAuthType
	}

	return nil
}
