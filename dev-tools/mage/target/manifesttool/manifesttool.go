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

package manifesttool

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

const manifestToolImage = "docker.elastic.co/infra/manifest-tool:latest"

// SupportedVersions is the definition of supported version files
type SupportedVersions struct {
	Variants []map[string]string `yaml:"variants"`
}

// ManifestTool are targets to build multi-platform images with manifest-tool
type ManifestTool mg.Namespace

// PushSupportedVersions pushes images for versions defined in supported-versions.yml files
func (m ManifestTool) PushSupportedVersions() error {
	if runtime.GOOS != "linux" {
		return errors.Errorf("pushing supported versions in '%s' is not supported. Only linux is supported at this moment", runtime.GOOS)
	}

	fmt.Println(">> manifest-tool: Pushing docker images for supported versions")
	return m.pushForEachVariant()
}

func (m ManifestTool) pushForEachVariant() error {
	files, err := findSupportedVersionsFiles()
	if err != nil {
		return errors.Wrap(err, "finding supported versions files")
	}

	for _, f := range files {
		err := forEachSupportedVersion(f)
		if err != nil {
			return errors.Wrapf(err, "pushing supported versions defined in %s", f)
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

func forEachSupportedVersion(file string) error {
	d, err := ioutil.ReadFile(file)
	if err != nil {
		return errors.Wrapf(err, "reading supported versions file %s", file)
	}

	module := filepath.Base(filepath.Dir(filepath.Dir(file)))

	moduleUppercase := strings.ToUpper(module)

	var supportedVersions SupportedVersions

	err = yaml.Unmarshal(d, &supportedVersions)
	if err != nil {
		return errors.Wrapf(err, "parsing supported versions file %s", file)
	}

	for _, variant := range supportedVersions.Variants {
		index := "1"
		var codename string
		var version string
		for k, v := range variant {
			switch k {
			case moduleUppercase + "_VERSION":
				version = v
			case moduleUppercase + "_CODENAME":
			case moduleUppercase + "_VARIANT":
				codename = v
			case moduleUppercase + "_INDEX":
				index = v
			default:
				codename = ""
				version = ""
				index = "1"
			}
		}

		var tag string
		if codename == "" {
			tag = fmt.Sprintf("%s-%s", version, index)
		} else {
			tag = fmt.Sprintf("%s-%s-%s", codename, version, index)
		}

		// supported platforms on CI: linux/amd64, linux/arm64
		platform := runtime.GOOS + "/" + runtime.GOARCH
		fmt.Printf(">> manifest-tool: Pushing images for module '%s', tag '%s' on platform '%s'\n", module, tag, platform)

		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		// this file path uses *NIX separator, because the images are supposed to be built under linux
		dockerConfigFile := homeDir + "/.docker/config.json"

		image := fmt.Sprintf("docker.elastic.co/integrations-ci/beats-%s:%s", module, tag)
		var stderr bytes.Buffer
		_, err = sh.Exec(
			map[string]string{}, nil, &stderr,
			"docker", "run", "--rm", "--mount", "src="+dockerConfigFile+",target=/docker-config,type=bind",
			manifestToolImage,
			"--platforms", platform,
			"--template", image,
			"--target", image,
		)
		if err != nil {
			io.Copy(os.Stderr, &stderr)
			return err
		}
	}
	fmt.Println(">> manifest-tool: OK")

	return nil
}
