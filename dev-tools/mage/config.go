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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/magefile/mage/mg"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Paths to generated config file templates.
var (
	shortTemplate     = filepath.Join("build", BeatName+".yml.tmpl")
	referenceTemplate = filepath.Join("build", BeatName+".reference.yml.tmpl")
	dockerTemplate    = filepath.Join("build", BeatName+".docker.yml.tmpl")
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
	Templates                []string // List of files or globs to load.
	ExtraVars                map[string]interface{}
	Short, Reference, Docker ConfigParams
}

type ConfigParams struct {
	Template string
	Deps     []interface{}
}

func DefaultConfigFileParams() ConfigFileParams {
	return ConfigFileParams{
		Templates: []string{LibbeatDir("_meta/config/*.tmpl")},
		ExtraVars: map[string]interface{}{},
		Short: ConfigParams{
			Template: LibbeatDir("_meta/config/default.short.yml.tmpl"),
		},
		Reference: ConfigParams{
			Template: LibbeatDir("_meta/config/default.reference.yml.tmpl"),
		},
		Docker: ConfigParams{
			Template: LibbeatDir("_meta/config/default.docker.yml.tmpl"),
		},
	}
}

// Config generates config files. Set DEV_OS and DEV_ARCH to change the target
// host for the generated configs. Defaults to linux/amd64.
func Config(types ConfigFileType, args ConfigFileParams, targetDir string) error {
	// Short
	if types.IsShort() {
		file := filepath.Join(targetDir, BeatName+".yml")
		if err := makeConfigTemplate(file, 0600, args, ShortConfigType); err != nil {
			return errors.Wrap(err, "failed making short config")
		}
	}

	// Reference
	if types.IsReference() {
		file := filepath.Join(targetDir, BeatName+".reference.yml")
		if err := makeConfigTemplate(file, 0644, args, ReferenceConfigType); err != nil {
			return errors.Wrap(err, "failed making reference config")
		}
	}

	// Docker
	if types.IsDocker() {
		file := filepath.Join(targetDir, BeatName+".docker.yml")
		if err := makeConfigTemplate(file, 0600, args, DockerConfigType); err != nil {
			return errors.Wrap(err, "failed making docker config")
		}
	}

	return nil
}

func makeConfigTemplate(destination string, mode os.FileMode, confParams ConfigFileParams, typ ConfigFileType) error {
	// Determine what type to build and set some parameters.
	var confFile ConfigParams
	var tmplParams map[string]interface{}
	switch typ {
	case ShortConfigType:
		confFile = confParams.Short
		tmplParams = map[string]interface{}{}
	case ReferenceConfigType:
		confFile = confParams.Reference
		tmplParams = map[string]interface{}{"Reference": true}
	case DockerConfigType:
		confFile = confParams.Docker
		tmplParams = map[string]interface{}{"Docker": true}
	default:
		panic(errors.Errorf("Invalid config file type: %v", typ))
	}

	// Build the dependencies.
	mg.SerialDeps(confFile.Deps...)

	// Set variables that are available in templates.
	// Rather than adding more "ExcludeX"/"UseX" options consider overwriting
	// one of the libbeat templates in your project by adding a file with the
	// same name to your _meta/config directory.
	params := map[string]interface{}{
		"GOOS":                           EnvOr("DEV_OS", "linux"),
		"GOARCH":                         EnvOr("DEV_ARCH", "amd64"),
		"BeatLicense":                    BeatLicense,
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
	params = joinMaps(params, confParams.ExtraVars, tmplParams)
	tmpl := template.New("config").Option("missingkey=error")
	funcs := joinMaps(FuncMap, template.FuncMap{
		"header":    header,
		"subheader": subheader,
		"indent":    indent,
		// include is necessary because you cannot pipe 'template' to a function
		// since 'template' is an action. This allows you to include a
		// template and indent it (e.g. {{ include "x.tmpl" . | indent 4 }}).
		"include": func(name string, data interface{}) (string, error) {
			buf := bytes.NewBuffer(nil)
			if err := tmpl.ExecuteTemplate(buf, name, data); err != nil {
				return "", err
			}
			return buf.String(), nil
		},
	})
	tmpl = tmpl.Funcs(funcs)

	fmt.Printf(">> Building %v for %v/%v\n", destination, params["GOOS"], params["GOARCH"])
	var err error
	for _, templateGlob := range confParams.Templates {
		if tmpl, err = tmpl.ParseGlob(templateGlob); err != nil {
			return errors.Wrapf(err, "failed to parse config templates in %q", templateGlob)
		}
	}

	data, err := ioutil.ReadFile(confFile.Template)
	if err != nil {
		return errors.Wrapf(err, "failed to read config template %q", confFile.Template)
	}

	tmpl, err = tmpl.Parse(string(data))
	if err != nil {
		return errors.Wrap(err, "failed to parse template")
	}

	out, err := os.OpenFile(CreateDir(destination), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if err = tmpl.Execute(out, EnvMap(params)); err != nil {
		return errors.Wrapf(err, "failed building %v", destination)
	}

	return nil
}

func header(title string) string {
	return makeHeading(title, "=")
}

func subheader(title string) string {
	return makeHeading(title, "-")
}

var nonWhitespaceRegex = regexp.MustCompile(`(?m)(^.*\S.*$)`)

// indent pads all non-whitespace lines with the number of spaces specified.
func indent(spaces int, content string) string {
	pad := strings.Repeat(" ", spaces)
	return nonWhitespaceRegex.ReplaceAllString(content, pad+"$1")
}

func makeHeading(title, separator string) string {
	const line = 80
	leftEquals := (line - len("# ") - len(title) - 2*len(" ")) / 2
	rightEquals := leftEquals + len(title)%2
	return "# " + strings.Repeat(separator, leftEquals) + " " + title + " " + strings.Repeat(separator, rightEquals)
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
