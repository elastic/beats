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

//go:build ((linux && go1.8) || (darwin && go1.10)) && cgo
// +build linux,go1.8 darwin,go1.10
// +build cgo

package plugin

import (
	"flag"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type pluginList struct {
	paths  []string
	logger *logp.Logger
}

func (p *pluginList) String() string {
	return strings.Join(p.paths, ",")
}

func (p *pluginList) Set(v string) error {
	for _, path := range p.paths {
		if path == v {
			p.logger.Warnf("%s is already a registered plugin", path)
			return nil
		}
	}
	p.paths = append(p.paths, v)
	return nil
}

var plugins = &pluginList{
	logger: logp.NewLogger("cli"),
}

func init() {
	flag.Var(plugins, "plugin", "Load additional plugins")
}

func Initialize() error {
	if len(plugins.paths) > 0 {
		cfgwarn.Experimental("loadable plugin support is experimental")
	}

	for _, path := range plugins.paths {
		plugins.logger.Infof("loading plugin bundle: %v", path)

		if err := LoadPlugins(path); err != nil {
			return err
		}
	}

	return nil
}
