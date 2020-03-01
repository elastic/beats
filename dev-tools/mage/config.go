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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/magefile/mage/mg"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Paths to generated config file templates.
var (
	shortTemplate     = filepath.Join("build", BeatName+".yml.tmpl")
	referenceTemplate = filepath.Join("build", BeatName+".reference.yml.tmpl")
	dockerTemplate    = filepath.Join("build", BeatName+".docker.yml.tmpl")

	defaultConfigFileParams = ConfigFileParams{
		ShortParts: []string{
			OSSBeatDir("_meta/beat.yml"),
			LibbeatDir("_meta/config.yml.tmpl"),
		},
		ReferenceParts: []string{
			OSSBeatDir("_meta/beat.reference.yml"),
			LibbeatDir("_meta/config.reference.yml.tmpl"),
		},
		DockerParts: []string{
			OSSBeatDir("_meta/beat.docker.yml"),
			LibbeatDir("_meta/config.docker.yml"),
		},
	}
)

// ConfigFileType is a bitset that indicates what types of config files to
// generate.
type ConfigFileType uint8

// Config file types.
const (
	ShortConfigType ConfigFileType = 1 << iota
	ReferenceConfigType
	DockerConfigType

	AllConfigTypes ConfigFileType = 0xFF
)

// IsShort return true if ShortConfigType is set.
func (t ConfigFileType) IsShort() bool { return t&ShortConfigType > 0 }

// IsReference return true if ReferenceConfigType is set.
func (t ConfigFileType) IsReference() bool { return t&ReferenceConfigType > 0 }

// IsDocker return true if DockerConfigType is set.
func (t ConfigFileType) IsDocker() bool { return t&DockerConfigType > 0 }

// ConfigFileParams defines the files that make up each config file.
type ConfigFileParams struct {
	ShortParts     []string // List of files or globs.
	ShortDeps      []interface{}
	ReferenceParts []string // List of files or globs.
	ReferenceDeps  []interface{}
	DockerParts    []string // List of files or globs.
	DockerDeps     []interface{}
	ExtraVars      map[string]interface{}
}

// Empty checks if configuration files are set.
func (c ConfigFileParams) Empty() bool {
	return len(c.ShortParts) == len(c.ReferenceDeps) && len(c.ReferenceParts) == len(c.DockerParts) && len(c.DockerParts) == 0
}

// Config generates config files. Set DEV_OS and DEV_ARCH to change the target
// host for the generated configs. Defaults to linux/amd64.
func Config(types ConfigFileType, args ConfigFileParams, targetDir string) error {
	if args.Empty() {
		args = defaultConfigFileParams
	}

	if err := makeConfigTemplates(types, args); err != nil {
		return errors.Wrap(err, "failed making config templates")
	}

	params := map[string]interface{}{
		"GOOS":                           EnvOr("DEV_OS", "linux"),
		"GOARCH":                         EnvOr("DEV_ARCH", "amd64"),
		"Reference":                      false,
		"Docker":                         false,
		"ExcludeConsole":                 false,
		"ExcludeFileOutput":              false,
		"ExcludeKafka":                   false,
		"ExcludeLogstash":                false,
		"ExcludeRedis":                   false,
		"UseObserverProcessor":           false,
		"UseDockerMetadataProcessor":     true,
		"UseKubernetesMetadataProcessor": false,
		"ExcludeDashboards":              false,
	}
	for k, v := range args.ExtraVars {
		params[k] = v
	}

	// Short
	if types.IsShort() {
		file := filepath.Join(targetDir, BeatName+".yml")
		fmt.Printf(">> Building %v for %v/%v\n", file, params["GOOS"], params["GOARCH"])
		if err := ExpandFile(shortTemplate, file, params); err != nil {
			return errors.Wrapf(err, "failed building %v", file)
		}
	}

	// Reference
	if types.IsReference() {
		file := filepath.Join(targetDir, BeatName+".reference.yml")
		params["Reference"] = true
		fmt.Printf(">> Building %v for %v/%v\n", file, params["GOOS"], params["GOARCH"])
		if err := ExpandFile(referenceTemplate, file, params); err != nil {
			return errors.Wrapf(err, "failed building %v", file)
		}
	}

	// Docker
	if types.IsDocker() {
		file := filepath.Join(targetDir, BeatName+".docker.yml")
		params["Reference"] = false
		params["Docker"] = true
		fmt.Printf(">> Building %v for %v/%v\n", file, params["GOOS"], params["GOARCH"])
		if err := ExpandFile(dockerTemplate, file, params); err != nil {
			return errors.Wrapf(err, "failed building %v", file)
		}
	}

	return nil
}

func makeConfigTemplates(types ConfigFileType, args ConfigFileParams) error {
	var err error

	if types.IsShort() {
		mg.SerialDeps(args.ShortDeps...)
		if err = makeConfigTemplate(shortTemplate, 0600, args.ShortParts...); err != nil {
			return err
		}
	}

	if types.IsReference() {
		mg.SerialDeps(args.ReferenceDeps...)
		if err = makeConfigTemplate(referenceTemplate, 0644, args.ReferenceParts...); err != nil {
			return err
		}
	}

	if types.IsDocker() {
		mg.SerialDeps(args.DockerDeps...)
		if err = makeConfigTemplate(dockerTemplate, 0600, args.DockerParts...); err != nil {
			return err
		}
	}

	return nil
}

func makeConfigTemplate(destination string, mode os.FileMode, parts ...string) error {
	configFiles, err := FindFiles(parts...)
	if err != nil {
		return errors.Wrap(err, "failed to find config templates")
	}

	if IsUpToDate(destination, configFiles...) {
		return nil
	}

	log.Println(">> Building", destination)
	if err = FileConcat(destination, mode, configFiles...); err != nil {
		return err
	}
	if err = FindReplace(destination, regexp.MustCompile("beatname"), "{{.BeatName}}"); err != nil {
		return err
	}
	return FindReplace(destination, regexp.MustCompile("beat-index-prefix"), "{{.BeatIndexPrefix}}")
}

const moduleConfigTemplate = `
#==========================  Modules configuration =============================
{{.BeatName}}.modules:
{{range $mod := .Modules}}
#{{$mod.Dashes}} {{$mod.Title | title}} Module {{$mod.Dashes}}
{{$mod.Config}}
{{- end}}

`

type moduleConfigTemplateData struct {
	ID     string
	Title  string
	Dashes string
	Config string
}

type moduleFieldsYmlData []struct {
	Title       string `json:"title"`
	ShortConfig bool   `json:"short_config"`
}

func readModuleFieldsYml(path string) (title string, useShort bool, err error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", false, err
	}

	var fd moduleFieldsYmlData
	if err = yaml.Unmarshal(data, &fd); err != nil {
		return "", false, err
	}

	if len(fd) == 0 {
		return "", false, errors.New("module not found in fields.yml")
	}

	return fd[0].Title, fd[0].ShortConfig, nil
}

// moduleDashes returns a string containing the correct number of dashes '-' to
// center the modules title in the middle of the line surrounded by an equal
// number of dashes on each side.
func moduleDashes(name string) string {
	const (
		lineLen        = 80
		headerLen      = len("#")
		titleSuffixLen = len(" Module ")
	)

	numDashes := lineLen - headerLen - titleSuffixLen - len(name) - 1
	numDashes /= 2
	return strings.Repeat("-", numDashes)
}

// GenerateModuleReferenceConfig generates a reference config file and includes
// modules found from the given module dirs.
func GenerateModuleReferenceConfig(out string, moduleDirs ...string) error {
	var moduleConfigs []moduleConfigTemplateData
	for _, dir := range moduleDirs {
		modules, err := ioutil.ReadDir(dir)
		if err != nil {
			return err
		}

		for _, modDirInfo := range modules {
			if !modDirInfo.IsDir() {
				continue
			}
			name := modDirInfo.Name()

			// Get title from fields.yml.
			title, _, err := readModuleFieldsYml(filepath.Join(dir, name, "_meta/fields.yml"))
			if err != nil {
				title = strings.Title(name)
			}

			// Prioritize config.reference.yml, but fallback to config.yml.
			files := []string{
				filepath.Join(dir, name, "_meta/config.reference.yml"),
				filepath.Join(dir, name, "_meta/config.yml"),
			}

			var data []byte
			for _, f := range files {
				data, err = ioutil.ReadFile(f)
				if err != nil {
					if os.IsNotExist(err) {
						continue
					}
					return err
				}

				break
			}
			if data == nil {
				continue
			}

			moduleConfigs = append(moduleConfigs, moduleConfigTemplateData{
				ID:     name,
				Title:  title,
				Dashes: moduleDashes(title),
				Config: string(data),
			})
		}
	}

	// Sort them by their module dir name, but put system first.
	sort.Slice(moduleConfigs, func(i, j int) bool {
		// Bubble system to the top of the list.
		if moduleConfigs[i].ID == "system" {
			return true
		} else if moduleConfigs[j].ID == "system" {
			return false
		}
		return moduleConfigs[i].ID < moduleConfigs[j].ID
	})

	config := MustExpand(moduleConfigTemplate, map[string]interface{}{
		"Modules": moduleConfigs,
	})

	return ioutil.WriteFile(createDir(out), []byte(config), 0644)
}
