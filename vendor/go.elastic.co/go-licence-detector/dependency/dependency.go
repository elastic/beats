// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package dependency // import "go.elastic.co/go-licence-detector/dependency"

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	securejoin "github.com/cyphar/filepath-securejoin"
)

// List holds direct and indirect dependency information.
type List struct {
	Direct   []Info
	Indirect []Info
}

// Info holds information about a dependency.
type Info struct {
	Name                    string `json:"name"`
	Dir                     string `json:"-"`
	LicenceFile             string `json:"licenceFile"`
	LicenceType             string `json:"licenceType"`
	URL                     string `json:"url"`
	Version                 string `json:"version"`
	VersionTime             string `json:"versionTime"`
	LicenceTextOverrideFile string `json:"licenceTextOverrideFile"`
}

// Overrides is a mapping from module name to dependency info.
type Overrides map[string]Info

// LoadOverrides loads the dependency overrides from the given file.
// LicenceTextOverrideFile will be read relative to the parent directory of the file.
func LoadOverrides(file string) (Overrides, error) {
	depMap := make(Overrides)
	if file == "" {
		return depMap, nil
	}

	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open overrides file %s: %w", file, err)
	}
	defer f.Close()

	rootDir, err := filepath.Abs(filepath.Dir(file))
	if err != nil {
		return nil, fmt.Errorf("failed to determine absolute path of overrides file: %w", err)
	}

	decoder := json.NewDecoder(f)
	for {
		var dep Info
		err := decoder.Decode(&dep)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return depMap, nil
			}

			return depMap, fmt.Errorf("error reading dependency information: %w", err)
		}

		if dep.LicenceTextOverrideFile != "" {
			licFile, err := securejoin.SecureJoin(rootDir, dep.LicenceTextOverrideFile)
			if err != nil {
				return nil, fmt.Errorf("failed to generate secure path to licence text file of %s: %w", dep.Name, err)
			}
			dep.LicenceFile = licFile
		}

		depMap[dep.Name] = dep
	}
}
