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

package mage

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const kibanaBuildDir = "build/kibana"

// KibanaDashboards collects the Kibana dashboards files and generates the
// index patterns based on the fields.yml file. It outputs to build/kibana.
// Use PackageKibanaDashboardsFromBuildDir() with this.
func KibanaDashboards(moduleDirs ...string) error {
	if err := os.RemoveAll(kibanaBuildDir); err != nil {
		return err
	}
	if err := os.MkdirAll(kibanaBuildDir, 0755); err != nil {
		return err
	}

	// Copy the OSS Beat's common dashboards if they exist. This assumes that
	// X-Pack Beats only add dashboards with modules (this will require a
	// change if we have X-Pack only Beats).
	cp := &CopyTask{Source: OSSBeatDir("_meta/kibana"), Dest: kibanaBuildDir}
	if err := cp.Execute(); err != nil && !os.IsNotExist(errors.Cause(err)) {
		return err
	}

	// Copy dashboards from modules.
	for _, dir := range moduleDirs {
		kibanaDirs, err := filepath.Glob(filepath.Join(dir, "*/_meta/kibana"))
		if err != nil {
			return err
		}

		for _, kibanaDir := range kibanaDirs {
			cp := &CopyTask{Source: kibanaDir, Dest: kibanaBuildDir}
			if err = cp.Execute(); err != nil {
				return err
			}
		}
	}

	return nil
}

// PackageKibanaDashboardsFromBuildDir reconfigures the packaging configuration
// to pull Kibana dashboards from build/kibana rather than _meta/kibana.generated.
// Use this with KibanaDashboards() (aka mage dashboards).
func PackageKibanaDashboardsFromBuildDir() {
	kibanaDashboards := PackageFile{
		Source: "build/kibana",
		Mode:   0644,
	}

	for _, pkgArgs := range Packages {
		for _, pkgType := range pkgArgs.Types {
			switch pkgType {
			case TarGz, Zip, Docker:
				pkgArgs.Spec.ReplaceFile("kibana", kibanaDashboards)
			case Deb, RPM:
				pkgArgs.Spec.ReplaceFile("/usr/share/{{.BeatName}}/kibana", kibanaDashboards)
			default:
				panic(errors.Errorf("unhandled package type: %v", pkgType))
			}
			break
		}
	}
}
