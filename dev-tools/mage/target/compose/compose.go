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

package compose

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	devtools "github.com/menderesk/beats/v7/dev-tools/mage"
)

// SupportedVersions is the definition of supported version files
type SupportedVersions struct {
	Variants []map[string]string `yaml:"variants"`
}

// Compose are targets to manage compose scenarios
type Compose mg.Namespace

// BuildSupportedVersions builds images for versions defined in supported-versions.yml files
func (c Compose) BuildSupportedVersions() error {
	fmt.Println(">> compose: Building docker images for supported versions")
	return c.composeForEachVariant("build", "Building images")
}

// PushSupportedVersions pushes images for versions defined in supported-versions.yml files
func (c Compose) PushSupportedVersions() error {
	fmt.Println(">> compose: Pushing docker images for supported versions")
	return c.composeForEachVariant("push", "Pushing images")
}

func (c Compose) composeForEachVariant(action, message string) error {
	files, err := findSupportedVersionsFiles()
	if err != nil {
		return errors.Wrap(err, "finding supported versions files")
	}

	virtualenv, err := devtools.PythonVirtualenv()
	if err != nil {
		return errors.Wrap(err, "configuring Python virtual environment")
	}

	composePath, err := devtools.LookVirtualenvPath(virtualenv, "docker-compose")
	if err != nil {
		return errors.Wrapf(err, "looking up docker-compose in virtual environment %s", virtualenv)
	}

	for _, f := range files {
		err := forEachSupportedVersion(composePath, f, action, message)
		if err != nil {
			return errors.Wrapf(err, "executing action '%s' for supported versions defined in %s", action, f)
		}
	}

	return nil
}

func findSupportedVersionsFiles() ([]string, error) {
	if f := os.Getenv("SUPPORTED_VERSIONS_FILE"); len(f) > 0 {
		return []string{f}, nil
	}

	if module := os.Getenv("MODULE"); len(module) > 0 {
		path := filepath.Join("module", module, "_meta/supported-versions.yml")
		return []string{path}, nil
	}

	if input := os.Getenv("INPUT"); len(input) > 0 {
		path := filepath.Join("input", input, "_meta/supported-versions.yml")
		return []string{path}, nil
	}

	return devtools.FindFilesRecursive(func(path string, _ os.FileInfo) bool {
		return filepath.Base(path) == "supported-versions.yml"
	})
}

func forEachSupportedVersion(composePath, file string, action string, message string) error {
	d, err := ioutil.ReadFile(file)
	if err != nil {
		return errors.Wrapf(err, "reading supported versions file %s", file)
	}

	var supportedVersions SupportedVersions

	err = yaml.Unmarshal(d, &supportedVersions)
	if err != nil {
		return errors.Wrapf(err, "parsing supported versions file %s", file)
	}

	composeYmlPath, err := findComposeYmlPath(filepath.Dir(file))
	if err != nil {
		return errors.Wrapf(err, "looking for docker-compose.yml")
	}

	fmt.Printf(">> compose: Using compose file %s\n", composeYmlPath)
	for _, variant := range supportedVersions.Variants {
		fmt.Printf(">> compose: %s for variant %+v\n", message, variant)

		var stderr bytes.Buffer
		_, err := sh.Exec(variant, nil, &stderr, composePath, "-f", composeYmlPath, action)
		if err != nil {
			io.Copy(os.Stderr, &stderr)
			return err
		}
	}
	fmt.Println(">> compose: OK")

	return nil
}

func findComposeYmlPath(dir string) (string, error) {
	path := dir
	for {
		if path == "/" {
			break
		}

		composePath := filepath.Join(path, "docker-compose.yml")
		if _, err := os.Stat(composePath); err == nil {
			return composePath, nil
		}
		path = filepath.Dir(path)
	}

	return "", fmt.Errorf("searching for docker-compose.yml starting on dir %s", dir)
}
