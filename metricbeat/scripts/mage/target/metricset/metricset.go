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

package metricset

import (
	"os"
	"path/filepath"

	"github.com/magefile/mage/sh"

	devtools "github.com/elastic/beats/v8/dev-tools/mage"
)

// CreateMetricset creates a new metricset.
//
// Required ENV variables:
// * MODULE: Name of the module
// * METRICSET: Name of the metricset
func CreateMetricset() error {
	ve, err := devtools.PythonVirtualenv()
	if err != nil {
		return err
	}
	python, err := devtools.LookVirtualenvPath(ve, "python")
	if err != nil {
		return err
	}
	beatsDir, err := devtools.ElasticBeatsDir()
	if err != nil {
		return err
	}
	scriptPath := filepath.Join(beatsDir, "metricbeat", "scripts", "create_metricset.py")

	_, err = sh.Exec(
		map[string]string{}, os.Stdout, os.Stderr, python, scriptPath,
		"--path", devtools.CWD(), "--es_beats", beatsDir,
		"--module", os.Getenv("MODULE"), "--metricset", os.Getenv("METRICSET"),
	)
	return err
}
