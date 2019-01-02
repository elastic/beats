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

	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"

	"github.com/elastic/beats/dev-tools/mage"
)

// GenerateMetricSet generates a new MetricSet. It will create the module too
// if it does not already exist.
func GenerateMetricSet() error {
	module, metricset := os.Getenv("MODULE"), os.Getenv("METRICSET")
	if module == "" {
		return errors.New("MODULE must be set")
	}
	if metricset == "" {
		return errors.New("METRICSET must be set")
	}

	ve, err := mage.PythonVirtualenv()
	if err != nil {
		return err
	}

	pythonPath, err := mage.LookVirtualenvPath(ve, "python")
	if err != nil {
		return err
	}

	elasticBeats, err := mage.ElasticBeatsDir()
	if err != nil {
		return err
	}

	// TODO: Port this script to Go.
	return sh.RunV(
		pythonPath,
		filepath.Join(elasticBeats, "metricbeat/scripts/create_metricset.py"),
		"--path", mage.CWD(),
		"--es_beats", elasticBeats,
		"--module", module,
		"--metricset", metricset,
	)
}
