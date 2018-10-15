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
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const packageSpecFile = "dev-tools/packaging/packages.yml"

// Packages defines the set of packages to be built when the package target is
// executed.
var Packages []OSPackageArgs

// UseCommunityBeatPackaging configures the package target to build packages for
// a community Beat.
func UseCommunityBeatPackaging() {
	beatsDir, err := ElasticBeatsDir()
	if err != nil {
		panic(err)
	}

	err = LoadNamedSpec("community_beat", filepath.Join(beatsDir, packageSpecFile))
	if err != nil {
		panic(err)
	}
}

// UseElasticBeatPackaging configures the package target to build packages for
// an Elastic Beat. This means it will generate two sets of packages -- one
// that is purely OSS under Apache 2.0 and one that is licensed under the
// Elastic License and may contain additional X-Pack features.
func UseElasticBeatPackaging() {
	beatsDir, err := ElasticBeatsDir()
	if err != nil {
		panic(err)
	}

	err = LoadNamedSpec("elastic_beat", filepath.Join(beatsDir, packageSpecFile))
	if err != nil {
		panic(err)
	}
}

// UseElasticBeatWithoutXPackPackaging configures the package target to build packages for
// an Elastic Beat. This means it will generate two sets of packages -- one
// that is purely OSS under Apache 2.0 and one that is licensed under the
// Elastic License and may contain additional X-Pack features.
//
// NOTE: This method doesn't use binaries produced in the x-pack folder, this is
// a temporary packaging target for projects that depends on beat but do have concrete x-pack
// binaries.
func UseElasticBeatWithoutXPackPackaging() {
	beatsDir, err := ElasticBeatsDir()
	if err != nil {
		panic(err)
	}

	err = LoadNamedSpec("elastic_beat_without_xpack", filepath.Join(beatsDir, packageSpecFile))
	if err != nil {
		panic(err)
	}
}

// LoadNamedSpec loads a packaging specification with the given name from the
// specified YAML file. name should be a sub-key of 'specs'.
func LoadNamedSpec(name, file string) error {
	specs, err := LoadSpecs(file)
	if err != nil {
		return errors.Wrap(err, "failed to load spec file")
	}

	packages, found := specs[name]
	if !found {
		return errors.Errorf("%v not found in package specs", name)
	}

	log.Printf("%v package spec loaded from %v", name, file)
	Packages = packages
	return nil
}

// LoadSpecs loads the packaging specifications from the specified YAML file.
func LoadSpecs(file string) (map[string][]OSPackageArgs, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read from spec file")
	}

	type PackageYAML struct {
		Specs map[string][]OSPackageArgs `yaml:"specs"`
	}

	var packages PackageYAML
	if err = yaml.Unmarshal(data, &packages); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal spec data")
	}

	return packages.Specs, nil
}
