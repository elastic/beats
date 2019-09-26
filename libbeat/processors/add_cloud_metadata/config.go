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

package add_cloud_metadata

import (
	"fmt"
	"time"
)

type config struct {
	Timeout   time.Duration `config:"timeout"`   // Amount of time to wait for responses from the metadata services.
	Overwrite bool          `config:"overwrite"` // Overwrite if cloud.* fields already exist.
	Providers providerList  `config:"providers"` // List of providers to probe
}

type providerList []string

const (
	// Default config
	defaultTimeout = 3 * time.Second

	// Default overwrite
	defaultOverwrite = false
)

func defaultConfig() config {
	return config{
		Timeout:   defaultTimeout,
		Overwrite: defaultOverwrite,
		Providers: nil, // enable all local-only providers by default
	}
}

func (c *config) Validate() error {
	// XXX: remove this check. A bug in go-ucfg prevents the correct validation
	// on providerList
	return c.Providers.Validate()
}

func (l providerList) Has(name string) bool {
	for _, elem := range l {
		if string(elem) == name {
			return true
		}
	}
	return false
}

func (l *providerList) Validate() error {
	if l == nil {
		return nil
	}

	for _, name := range *l {
		if _, ok := cloudMetaProviders[name]; !ok {
			return fmt.Errorf("unknown provider '%v'", name)
		}
	}
	return nil

}
