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

/*
Package fileset contains the code that loads Filebeat modules (which are
composed of filesets).
*/

package fileset

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	errw "github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	mlimporter "github.com/elastic/beats/libbeat/ml-importer"
)

// Fileset struct is the representation of a fileset.
type Fileset struct {
	name        string
	mcfg        *ModuleConfig
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
	mcfg *ModuleConfig,
	fcfg *FilesetConfig) (*Fileset, error) {

	modulePath := filepath.Join(modulesPath, mcfg.Module)
	if _, err := os.Stat(modulePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Module %s (%s) doesn't exist.", mcfg.Module, modulePath)
	}

	return &Fileset{
		name:       name,
		mcfg:       mcfg,
		fcfg:       fcfg,
		modulePath: modulePath,
	}, nil
}

// String returns the module and the name of the fileset.
func (fs *Fileset) String() string {
	return fs.mcfg.Module + "/" + fs.name
}

// Read reads the manifest file and evaluates the variables.
func (fs *Fileset) Read(beatVersion string) error {
	var err error
	fs.manifest, err = fs.readManifest()
	if err != nil {
		return err
	}

	fs.vars, err = fs.evaluateVars(beatVersion)
	if err != nil {
		return err
	}

	fs.pipelineIDs, err = fs.getPipelineIDs(beatVersion)
	if err != nil {
		return err
	}

	return nil
}

// manifest structure is the representation of the manifest.yml file from the
// fileset.
type manifest struct {
	ModuleVersion   string                   `config:"module_version"`
	Vars            []map[string]interface{} `config:"var"`
	IngestPipeline  []string                 `config:"ingest_pipeline"`
	Input           string                   `config:"input"`
	MachineLearning []struct {
		Name       string `config:"name"`
		Job        string `config:"job"`
		Datafeed   string `config:"datafeed"`
		MinVersion string `config:"min_version"`
	} `config:"machine_learning"`
	Requires struct {
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
func (fs *Fileset) evaluateVars(beatVersion string) (map[string]interface{}, error) {
	var err error
	vars := map[string]interface{}{}
	vars["builtin"], err = fs.getBuiltinVars(beatVersion)
	if err != nil {
		return nil, err
	}

	for _, vals := range fs.manifest.Vars {
		var exists bool
		name, exists := vals["name"].(string)
		if !exists {
			return nil, fmt.Errorf("Variable doesn't have a string 'name' key")
		}

		value, exists := vals["default"]
		if !exists {
			return nil, fmt.Errorf("Variable %s doesn't have a 'default' key", name)
		}

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
		return applyTemplate(vars, v, false)
	case []interface{}:
		transformed := []interface{}{}
		for _, val := range v {
			s, ok := val.(string)
			if ok {
				transf, err := applyTemplate(vars, s, false)
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

// applyTemplate applies a Golang text/template. If specialDelims is set to true,
// the delimiters are set to `{<` and `>}` instead of `{{` and `}}`. These are easier to use
// in pipeline definitions.
func applyTemplate(vars map[string]interface{}, templateString string, specialDelims bool) (string, error) {
	tpl := template.New("text")
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
		"IngestPipeline": func(shortID string) string {
			return formatPipelineID(
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
func (fs *Fileset) getBuiltinVars(beatVersion string) (map[string]interface{}, error) {
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
		"hostname":    hostname,
		"domain":      domain,
		"module":      fs.mcfg.Module,
		"fileset":     fs.name,
		"beatVersion": beatVersion,
	}, nil
}

func (fs *Fileset) getInputConfig() (*common.Config, error) {
	path, err := applyTemplate(fs.vars, fs.manifest.Input, false)
	if err != nil {
		return nil, fmt.Errorf("Error expanding vars on the input path: %v", err)
	}
	contents, err := ioutil.ReadFile(filepath.Join(fs.modulePath, fs.name, path))
	if err != nil {
		return nil, fmt.Errorf("Error reading input file %s: %v", path, err)
	}

	yaml, err := applyTemplate(fs.vars, string(contents), false)
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
		cfg, err = common.MergeConfigs(cfg, overrides)
		if err != nil {
			return nil, fmt.Errorf("Error applying config overrides: %v", err)
		}
	}

	// force our pipeline ID
	rootPipelineID := ""
	if len(fs.pipelineIDs) > 0 {
		rootPipelineID = fs.pipelineIDs[0]
	}
	err = cfg.SetString("pipeline", -1, rootPipelineID)
	if err != nil {
		return nil, fmt.Errorf("Error setting the pipeline ID in the input config: %v", err)
	}

	// force our the module/fileset name
	err = cfg.SetString("_module_name", -1, fs.mcfg.Module)
	if err != nil {
		return nil, fmt.Errorf("Error setting the _module_name cfg in the input config: %v", err)
	}
	err = cfg.SetString("_fileset_name", -1, fs.name)
	if err != nil {
		return nil, fmt.Errorf("Error setting the _fileset_name cfg in the input config: %v", err)
	}

	cfg.PrintDebugf("Merged input config for fileset %s/%s", fs.mcfg.Module, fs.name)

	return cfg, nil
}

// getPipelineIDs returns the Ingest Node pipeline IDs
func (fs *Fileset) getPipelineIDs(beatVersion string) ([]string, error) {
	var pipelineIDs []string
	for _, ingestPipeline := range fs.manifest.IngestPipeline {
		path, err := applyTemplate(fs.vars, ingestPipeline, false)
		if err != nil {
			return nil, fmt.Errorf("Error expanding vars on the ingest pipeline path: %v", err)
		}

		pipelineIDs = append(pipelineIDs, formatPipelineID(fs.mcfg.Module, fs.name, path, beatVersion))
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
		path, err := applyTemplate(fs.vars, ingestPipeline, false)
		if err != nil {
			return nil, fmt.Errorf("Error expanding vars on the ingest pipeline path: %v", err)
		}

		strContents, err := ioutil.ReadFile(filepath.Join(fs.modulePath, fs.name, path))
		if err != nil {
			return nil, fmt.Errorf("Error reading pipeline file %s: %v", path, err)
		}

		encodedString, err := applyTemplate(vars, string(strContents), true)
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
			newContent, err := fixYAMLMaps(content)
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

// This function recursively converts maps with interface{} keys, as returned by
// yaml.Unmarshal, to maps of string keys, as expected by the json encoder
// that will be used when delivering the pipeline to Elasticsearch.
// Will return an error when something other than a string is used as a key.
func fixYAMLMaps(elem interface{}) (_ interface{}, err error) {
	switch v := elem.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, value := range v {
			keyS, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("key '%v' is not string but %T", key, key)
			}
			if result[keyS], err = fixYAMLMaps(value); err != nil {
				return nil, err
			}
		}
		return result, nil
	case map[string]interface{}:
		for key, value := range v {
			if v[key], err = fixYAMLMaps(value); err != nil {
				return nil, err
			}
		}
	case []interface{}:
		for idx, value := range v {
			if v[idx], err = fixYAMLMaps(value); err != nil {
				return nil, err
			}
		}
	}
	return elem, nil
}

// formatPipelineID generates the ID to be used for the pipeline ID in Elasticsearch
func formatPipelineID(module, fileset, path, beatVersion string) string {
	return fmt.Sprintf("filebeat-%s-%s-%s-%s", beatVersion, module, fileset, removeExt(filepath.Base(path)))
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

// GetMLConfigs returns the list of machine-learning configurations declared
// by this fileset.
func (fs *Fileset) GetMLConfigs() []mlimporter.MLConfig {
	var mlConfigs []mlimporter.MLConfig
	for _, ml := range fs.manifest.MachineLearning {
		mlConfigs = append(mlConfigs, mlimporter.MLConfig{
			ID:           fmt.Sprintf("filebeat-%s-%s-%s_ecs", fs.mcfg.Module, fs.name, ml.Name),
			JobPath:      filepath.Join(fs.modulePath, fs.name, ml.Job),
			DatafeedPath: filepath.Join(fs.modulePath, fs.name, ml.Datafeed),
			MinVersion:   ml.MinVersion,
		})
	}
	return mlConfigs
}
