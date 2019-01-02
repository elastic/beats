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
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type moduleOptions struct {
	Enable     map[string]struct{}
	ExtraVars  map[string]interface{}
	InputGlobs []string
	OutputDir  string
}

// ModuleOption is an option for control build behavior w.r.t. modules.
type ModuleOption func(params *moduleOptions)

// EnableModule enables the module with the given name (if found).
func EnableModule(name string) ModuleOption {
	return func(params *moduleOptions) {
		if params.Enable == nil {
			params.Enable = map[string]struct{}{}
		}
		params.Enable[name] = struct{}{}
	}
}

// SetTemplateVariable sets a key/value pair that will be available with
// rendering a config template.
func SetTemplateVariable(key string, value interface{}) ModuleOption {
	return func(params *moduleOptions) {
		if params.ExtraVars == nil {
			params.ExtraVars = map[string]interface{}{}
		}
		params.ExtraVars[key] = value
	}
}

// OutputDir specifies the directory where the output will be written.
func OutputDir(outputDir string) ModuleOption {
	return func(params *moduleOptions) {
		params.OutputDir = outputDir
	}
}

// InputGlobs is a list of globs to use when looking for files.
func InputGlobs(inputGlobs ...string) ModuleOption {
	return func(params *moduleOptions) {
		params.InputGlobs = inputGlobs
	}
}

var modulesDConfigTemplate = `
# Module: {{.Module}}
# Docs: https://www.elastic.co/guide/en/beats/{{.BeatName}}/{{ beat_doc_branch }}/{{.BeatName}}-module-{{.Module}}.html

{{.Config}}`[1:]

// GenerateDirModulesD generates a modules.d directory containing the
// <module>.yml.disabled files. It adds a header to each file containing a
// link to the documentation.
func GenerateDirModulesD(opts ...ModuleOption) error {
	args := moduleOptions{
		OutputDir:  "modules.d",
		InputGlobs: []string{"module/*/_meta/config.yml"},
	}
	for _, f := range opts {
		f(&args)
	}

	if err := os.RemoveAll(args.OutputDir); err != nil {
		return err
	}

	shortConfigs, err := FindFiles(args.InputGlobs...)
	if err != nil {
		return err
	}

	for _, f := range shortConfigs {
		parts := strings.Split(filepath.ToSlash(f), "/")
		if len(parts) < 2 {
			continue
		}
		moduleName := parts[1]

		config, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}

		params := map[string]interface{}{
			"GOOS":      EnvOr("DEV_OS", "linux"),
			"GOARCH":    EnvOr("DEV_ARCH", "amd64"),
			"Reference": false,
			"Docker":    false,
		}
		for k, v := range args.ExtraVars {
			params[k] = v
		}
		expandedConfig, err := Expand(string(config), params)
		if err != nil {
			return errors.Wrapf(err, "failed expanding config file=%v", f)
		}

		data, err := Expand(modulesDConfigTemplate, map[string]interface{}{
			"Module": moduleName,
			"Config": string(expandedConfig),
		})
		if err != nil {
			return err
		}

		target := filepath.Join(args.OutputDir, moduleName)
		if _, enabled := args.Enable[moduleName]; enabled {
			target += ".yml"
		} else {
			target += ".yml.disabled"
		}

		err = ioutil.WriteFile(CreateDir(target), []byte(data), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}
