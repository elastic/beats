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

package linux

import (
	"os"
	"path/filepath"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/system"
)

func init() {
	// Register the ModuleFactory function for the "system" module.
	if err := mb.Registry.AddModule("linux", NewModule); err != nil {
		panic(err)
	}
}

type LinuxModule interface {
	GetHostFS() string
}

// Module defines the base module config used in `linux`
type Module struct {
	mb.BaseModule
	HostFS  string `config:"hostfs"`
	UserSet bool
	Period  time.Duration
}

// NewModule initializes a new module
func NewModule(base mb.BaseModule) (mb.Module, error) {
	// This only needs to be configured once for all system modules.

	config := struct {
		Hostfs string        `config:"hostfs"`
		Period time.Duration `config:"period"`
	}{}

	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}
	userSet := false
	dir := config.Hostfs
	if dir == "" {
		dir = "/"
	} else {
		userSet = true
	}

	return &Module{BaseModule: base, HostFS: dir, Period: config.Period, UserSet: userSet}, nil
}

// In the case of a few vendored libraries, we need to set hostfs globally.
// On the off chance that the user is doing something particularly weird
func trySetHostfsEnv(path string) {
	// Making a decision here to treat the linux module as secondary to the system module.
	_, isSet := os.LookupEnv("HOST_PROC")
	if isSet {
		return
	}

	system.InitModule(path)

}

func (m Module) ResolveHostFS(path string) string {
	return filepath.Join(m.HostFS, path)
}

func (m Module) IsSet() bool {
	return m.IsSet()
}
