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

// Package fileset contains the code that loads Filebeat modules (which are
// composed of filesets).
package fileset

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"text/template"

	"github.com/menderesk/go-ucfg"

	errw "github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/cfgwarn"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

// Fileset struct is the representation of a fileset.
type Fileset struct {
	name        string
	mname       string
	fcfg        *FilesetConfig
	modulePath  string
	manifest    *manifest
	vars        map[string]interface{}
	pipelineIDs []string
}

type pipeline struct {
	id       string
	contents map[string]interface{}
}

// New allocates a new Fileset object with the given configuration.
func New(
	modulesPath string,
	name string,
	mname string,
	fcfg *FilesetConfig) (*Fileset, error,
) {
	modulePath := filepath.Join(modulesPath, mname)
	if _, err := os.Stat(modulePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("module %s (%s) doesn't exist", mname, modulePath)
	}

	return &Fileset{
		name:       name,
		mname:      mname,
		fcfg:       fcfg,
		modulePath: modulePath,
	}, nil
}

// String returns the module and the name of the fileset.
func (fs *Fileset) String() string {
	return fs.mname + "/" + fs.name
}

// Read reads the manifest file and evaluates the variables.
func (fs *Fileset) Read(info beat.Info) error {
	var err error
	fs.manifest, err = fs.readManifest()
	if err != nil {
		return err
	}

	fs.vars, err = fs.evaluateVars(info)
	if err != nil {
		return err
	}

	fs.pipelineIDs, err = fs.getPipelineIDs(info)
	if err != nil {
		return err
	}

	return nil
}

// manifest structure is the representation of the manifest.yml file from the
// fileset.
type manifest struct {
	ModuleVersion  string                   `config:"module_version"`
	Vars           []map[string]interface{} `config:"var"`
	IngestPipeline []string                 `config:"ingest_pipeline"`
	Input          string                   `config:"input"`
	Requires       struct {
		Processors []ProcessorRequirement `config:"processors"`
	} `config:"requires"`
}

func newManifest(cfg *common.Config) (*manifest, error) {
	if err := cfgwarn.CheckRemoved6xSetting(cfg, "prospector"); err != nil {
		return nil, err
	}

	var manifest manifest
	err := cfg.Unpack(&manifest)
	if err != nil {
		return nil, err
	}

	return &manifest, nil
}

// ProcessorRequirement represents the declaration of a dependency to a particular
// Ingest Node processor / plugin.
type ProcessorRequirement struct {
	Name   string `config:"name"`
	Plugin string `config:"plugin"`
}

// readManifest reads the manifest file of the fileset.
func (fs *Fileset) readManifest() (*manifest, error) {
	cfg, err := common.LoadFile(filepath.Join(fs.modulePath, fs.name, "manifest.yml"))
	if err != nil {
		return nil, fmt.Errorf("Error reading manifest file: %v", err)
	}
	manifest, err := newManifest(cfg)
	if err != nil {
		return nil, fmt.Errorf("Error unpacking manifest: %v", err)
	}
	return manifest, nil
}

// evaluateVars resolves the fileset variables.
func (fs *Fileset) evaluateVars(info beat.Info) (map[string]interface{}, error) {
	var err error
	vars := map[string]interface{}{}
	vars["builtin"], err = fs.getBuiltinVars(info)
	if err != nil {
		return nil, err
	}

	for _, vals := range fs.manifest.Vars {
		var exists bool
		name, exists := vals["name"].(string)
		if !exists {
			return nil, fmt.Errorf("Variable doesn't have a string 'name' key")
		}

		// Variables are not required to have a default. Templates should
		// handle null default values as necessary.
		value := vals["default"]

		// evaluate OS specific vars
		osVals, exists := vals["os"].(map[string]interface{})
		if exists {
			osVal, exists := osVals[runtime.GOOS]
			if exists {
				value = osVal
			}
		}

		vars[name], err = resolveVariable(vars, value)
		if err != nil {
			return nil, fmt.Errorf("Error resolving variables on %s: %v", name, err)
		}
	}

	// overrides from the config
	for name, val := range fs.fcfg.Var {
		vars[name] = val
	}

	return vars, nil
}

// turnOffElasticsearchVars re-evaluates the variables that have `min_elasticsearch_version`
// set.
func (fs *Fileset) turnOffElasticsearchVars(vars map[string]interface{}, esVersion common.Version) (map[string]interface{}, error) {
	retVars := map[string]interface{}{}
	for key, val := range vars {
		retVars[key] = val
	}

	if !esVersion.IsValid() {
		return vars, errors.New("Unknown Elasticsearch version")
	}

	for _, vals := range fs.manifest.Vars {
		var ok bool
		name, ok := vals["name"].(string)
		if !ok {
			return nil, fmt.Errorf("Variable doesn't have a string 'name' key")
		}

		minESVersion, ok := vals["min_elasticsearch_version"].(map[string]interface{})
		if ok {
			minVersion, err := common.NewVersion(minESVersion["version"].(string))
			if err != nil {
				return vars, fmt.Errorf("Error parsing version %s: %v", minESVersion["version"].(string), err)
			}

			logp.Debug("fileset", "Comparing ES version %s with requirement of %s", esVersion.String(), minVersion)

			if esVersion.LessThan(minVersion) {
				retVars[name] = minESVersion["value"]
				logp.Info("Setting var %s (%s) to %v because Elasticsearch version is %s", name, fs, minESVersion["value"], esVersion.String())
			}
		}
	}

	return retVars, nil
}

// resolveVariable considers the value as a template so it can refer to built-in variables
// as well as other variables defined before them.
func resolveVariable(vars map[string]interface{}, value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return ApplyTemplate(vars, v, false)
	case []interface{}:
		transformed := []interface{}{}
		for _, val := range v {
			s, ok := val.(string)
			if ok {
				transf, err := ApplyTemplate(vars, s, false)
				if err != nil {
					return nil, fmt.Errorf("array: %v", err)
				}
				transformed = append(transformed, transf)
			} else {
				transformed = append(transformed, val)
			}
		}
		return transformed, nil
	}
	return value, nil
}

// ApplyTemplate applies a Golang text/template. If specialDelims is set to true,
// the delimiters are set to `{<` and `>}` instead of `{{` and `}}`. These are easier to use
// in pipeline definitions.
func ApplyTemplate(vars map[string]interface{}, templateString string, specialDelims bool) (string, error) {
	tpl := template.New("text").Option("missingkey=error")
	if specialDelims {
		tpl = tpl.Delims("{<", ">}")
	}

	tplFunctions, err := getTemplateFunctions(vars)
	if err != nil {
		return "", errw.Wrap(err, "error fetching template functions")
	}
	tpl = tpl.Funcs(tplFunctions)

	tpl, err = tpl.Parse(templateString)
	if err != nil {
		return "", fmt.Errorf("Error parsing template %s: %v", templateString, err)
	}
	buf := bytes.NewBufferString("")
	err = tpl.Execute(buf, vars)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func getTemplateFunctions(vars map[string]interface{}) (template.FuncMap, error) {
	builtinVars, ok := vars["builtin"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error fetching built-in vars as a dictionary")
	}

	return template.FuncMap{
		"inList": func(collection []interface{}, item string) bool {
			for _, h := range collection {
				if reflect.DeepEqual(item, h) {
					return true
				}
			}
			return false
		},
		"tojson": func(v interface{}) (string, error) {
			var buf strings.Builder
			enc := json.NewEncoder(&buf)
			enc.SetEscapeHTML(false)
			err := enc.Encode(v)
			return buf.String(), err
		},
		"IngestPipeline": func(shortID string) string {
			return FormatPipelineID(
				builtinVars["prefix"].(string),
				builtinVars["module"].(string),
				builtinVars["fileset"].(string),
				shortID,
				builtinVars["beatVersion"].(string),
			)
		},
	}, nil
}

// getBuiltinVars computes the supported built in variables and groups them
// in a dictionary
func (fs *Fileset) getBuiltinVars(info beat.Info) (map[string]interface{}, error) {
	host, err := os.Hostname()
	if err != nil || len(host) == 0 {
		return nil, fmt.Errorf("Error getting the hostname: %v", err)
	}
	split := strings.SplitN(host, ".", 2)
	hostname := split[0]
	domain := ""
	if len(split) > 1 {
		domain = split[1]
	}

	return map[string]interface{}{
		"prefix":      info.IndexPrefix,
		"hostname":    hostname,
		"domain":      domain,
		"module":      fs.mname,
		"fileset":     fs.name,
		"beatVersion": info.Version,
	}, nil
}

func (fs *Fileset) getInputConfig() (*common.Config, error) {
	path, err := ApplyTemplate(fs.vars, fs.manifest.Input, false)
	if err != nil {
		return nil, fmt.Errorf("Error expanding vars on the input path: %v", err)
	}
	contents, err := ioutil.ReadFile(filepath.Join(fs.modulePath, fs.name, path))
	if err != nil {
		return nil, fmt.Errorf("Error reading input file %s: %v", path, err)
	}

	yaml, err := ApplyTemplate(fs.vars, string(contents), false)
	if err != nil {
		return nil, fmt.Errorf("Error interpreting the template of the input: %v", err)
	}

	cfg, err := common.NewConfigWithYAML([]byte(yaml), "")
	if err != nil {
		return nil, fmt.Errorf("Error reading input config: %v", err)
	}

	cfg, err = mergePathDefaults(cfg)
	if err != nil {
		return nil, err
	}

	// overrides
	if len(fs.fcfg.Input) > 0 {
		overrides, err := common.NewConfigFrom(fs.fcfg.Input)
		if err != nil {
			return nil, fmt.Errorf("Error creating config from input overrides: %v", err)
		}
		cfg, err = common.MergeConfigsWithOptions([]*common.Config{cfg, overrides}, ucfg.FieldReplaceValues("**.paths"), ucfg.FieldAppendValues("**.processors"))
		if err != nil {
			return nil, fmt.Errorf("Error applying config overrides: %v", err)
		}
	}

	const pipelineField = "pipeline"
	if !cfg.HasField(pipelineField) {
		rootPipelineID := ""
		if len(fs.pipelineIDs) > 0 {
			rootPipelineID = fs.pipelineIDs[0]
		}
		if err := cfg.SetString(pipelineField, -1, rootPipelineID); err != nil {
			return nil, errw.Wrap(err, "error setting the fileset pipeline ID in config")
		}
	}

	// force our the module/fileset name
	err = cfg.SetString("_module_name", -1, fs.mname)
	if err != nil {
		return nil, fmt.Errorf("Error setting the _module_name cfg in the input config: %v", err)
	}
	err = cfg.SetString("_fileset_name", -1, fs.name)
	if err != nil {
		return nil, fmt.Errorf("Error setting the _fileset_name cfg in the input config: %v", err)
	}

	cfg.PrintDebugf("Merged input config for fileset %s/%s", fs.mname, fs.name)

	return cfg, nil
}

// getPipelineIDs returns the Ingest Node pipeline IDs
func (fs *Fileset) getPipelineIDs(info beat.Info) ([]string, error) {
	var pipelineIDs []string
	for _, ingestPipeline := range fs.manifest.IngestPipeline {
		path, err := ApplyTemplate(fs.vars, ingestPipeline, false)
		if err != nil {
			return nil, fmt.Errorf("Error expanding vars on the ingest pipeline path: %v", err)
		}

		pipelineIDs = append(pipelineIDs, FormatPipelineID(info.IndexPrefix, fs.mname, fs.name, path, info.Version))
	}

	return pipelineIDs, nil
}

// GetPipelines returns the JSON content of the Ingest Node pipeline that parses the logs.
func (fs *Fileset) GetPipelines(esVersion common.Version) (pipelines []pipeline, err error) {
	vars, err := fs.turnOffElasticsearchVars(fs.vars, esVersion)
	if err != nil {
		return nil, err
	}

	for idx, ingestPipeline := range fs.manifest.IngestPipeline {
		path, err := ApplyTemplate(fs.vars, ingestPipeline, false)
		if err != nil {
			return nil, fmt.Errorf("Error expanding vars on the ingest pipeline path: %v", err)
		}

		strContents, err := ioutil.ReadFile(filepath.Join(fs.modulePath, fs.name, path))
		if err != nil {
			return nil, fmt.Errorf("Error reading pipeline file %s: %v", path, err)
		}

		encodedString, err := ApplyTemplate(vars, string(strContents), true)
		if err != nil {
			return nil, fmt.Errorf("Error interpreting the template of the ingest pipeline: %v", err)
		}

		var content map[string]interface{}
		switch extension := strings.ToLower(filepath.Ext(path)); extension {
		case ".json":
			if err = json.Unmarshal([]byte(encodedString), &content); err != nil {
				return nil, fmt.Errorf("Error JSON decoding the pipeline file: %s: %v", path, err)
			}
		case ".yaml", ".yml":
			if err = yaml.Unmarshal([]byte(encodedString), &content); err != nil {
				return nil, fmt.Errorf("Error YAML decoding the pipeline file: %s: %v", path, err)
			}
			newContent, err := FixYAMLMaps(content)
			if err != nil {
				return nil, fmt.Errorf("Failed to sanitize the YAML pipeline file: %s: %v", path, err)
			}
			content = newContent.(map[string]interface{})
		default:
			return nil, fmt.Errorf("Unsupported extension '%s' for pipeline file: %s", extension, path)
		}

		pipelineID := fs.pipelineIDs[idx]

		p := pipeline{
			pipelineID,
			content,
		}
		pipelines = append(pipelines, p)
	}

	return pipelines, nil
}

// FixYAMLMaps recursively converts maps with interface{} keys, as returned by
// yaml.Unmarshal, to maps of string keys, as expected by the json encoder
// that will be used when delivering the pipeline to Elasticsearch.
// Will return an error when something other than a string is used as a key.
func FixYAMLMaps(elem interface{}) (_ interface{}, err error) {
	switch v := elem.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, value := range v {
			keyS, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("key '%v' is not string but %T", key, key)
			}
			if result[keyS], err = FixYAMLMaps(value); err != nil {
				return nil, err
			}
		}
		return result, nil
	case map[string]interface{}:
		for key, value := range v {
			if v[key], err = FixYAMLMaps(value); err != nil {
				return nil, err
			}
		}
	case []interface{}:
		for idx, value := range v {
			if v[idx], err = FixYAMLMaps(value); err != nil {
				return nil, err
			}
		}
	}
	return elem, nil
}

// FormatPipelineID generates the ID to be used for the pipeline ID in Elasticsearch
func FormatPipelineID(prefix, module, fileset, path, version string) string {
	if module == "" && fileset == "" {
		return fmt.Sprintf("%s-%s-%s", prefix, version, removeExt(filepath.Base(path)))
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s", prefix, version, module, fileset, removeExt(filepath.Base(path)))
}

// removeExt returns the file name without the extension. If no dot is found,
// returns the same as the input.
func removeExt(path string) string {
	for i := len(path) - 1; i >= 0 && !os.IsPathSeparator(path[i]); i-- {
		if path[i] == '.' {
			return path[:i]
		}
	}
	return path
}

// GetRequiredProcessors returns the list of processors on which this
// fileset depends.
func (fs *Fileset) GetRequiredProcessors() []ProcessorRequirement {
	return fs.manifest.Requires.Processors
}
