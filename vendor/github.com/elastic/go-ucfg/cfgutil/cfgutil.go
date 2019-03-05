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

package cfgutil

import "github.com/elastic/go-ucfg"

// Collector collects and merges multiple generated *ucfg.Config, remembering
// errors, for postponing error checking after having merged all loaded configurations.
type Collector struct {
	config *ucfg.Config
	err    error
	opts   []ucfg.Option
}

func NewCollector(cfg *ucfg.Config, opts ...ucfg.Option) *Collector {
	if cfg == nil {
		cfg = ucfg.New()
	}
	return &Collector{config: cfg, err: nil}
}

func (c *Collector) GetOptions() []ucfg.Option {
	return c.opts
}

func (c *Collector) Get() (*ucfg.Config, error) {
	return c.config, c.err
}

func (c *Collector) Config() *ucfg.Config {
	return c.config
}

func (c *Collector) Error() error {
	return c.err
}

func (c *Collector) Add(cfg *ucfg.Config, err error) error {
	if c.err != nil {
		return c.err
	}

	if err != nil {
		c.err = err
		return err
	}

	if cfg != nil {
		err = c.config.Merge(cfg, c.opts...)
		if err != nil {
			c.err = err
		}
	}
	return err
}
