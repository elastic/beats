/*
Package fileset contains the code that loads Filebeat modules (which are
composed of filesets).
*/

package fileset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	mlimporter "github.com/elastic/beats/libbeat/ml-importer"
)

// Fileset struct is the representation of a fileset.
type Fileset struct {
	name       string
	mcfg       *ModuleConfig
	fcfg       *FilesetConfig
	modulePath string
	manifest   *manifest
	vars       map[string]interface{}
	pipelineID string
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

	fs.vars, err = fs.evaluateVars()
	if err != nil {
		return err
	}

	fs.pipelineID, err = fs.getPipelineID(beatVersion)
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
	IngestPipeline  string                   `config:"ingest_pipeline"`
	Input           string                   `config:"input"`
	Prospector      string                   `config:"prospector"`
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
	var manifest manifest
	err := cfg.Unpack(&manifest)
	if err != nil {
		return nil, err
	}
	if manifest.Prospector != "" {
		manifest.Input = manifest.Prospector
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
func (fs *Fileset) evaluateVars() (map[string]interface{}, error) {
	var err error
	vars := map[string]interface{}{}
	vars["builtin"], err = fs.getBuiltinVars()
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
func (fs *Fileset) turnOffElasticsearchVars(vars map[string]interface{}, esVersion string) (map[string]interface{}, error) {
	retVars := map[string]interface{}{}
	for key, val := range vars {
		retVars[key] = val
	}

	haveVersion, err := common.NewVersion(esVersion)
	if err != nil {
		return vars, fmt.Errorf("Error parsing version %s: %v", esVersion, err)
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

			logp.Debug("fileset", "Comparing ES version %s with requirement of %s", haveVersion, minVersion)

			if haveVersion.LessThan(minVersion) {
				retVars[name] = minESVersion["value"]
				logp.Info("Setting var %s (%s) to %v because Elasticsearch version is %s", name, fs, minESVersion["value"], haveVersion)
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
	tpl, err := tpl.Parse(templateString)
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

// getBuiltinVars computes the supported built in variables and groups them
// in a dictionary
func (fs *Fileset) getBuiltinVars() (map[string]interface{}, error) {
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
		"hostname": hostname,
		"domain":   domain,
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
	err = cfg.SetString("pipeline", -1, fs.pipelineID)
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

// getPipelineID returns the Ingest Node pipeline ID
func (fs *Fileset) getPipelineID(beatVersion string) (string, error) {
	path, err := applyTemplate(fs.vars, fs.manifest.IngestPipeline, false)
	if err != nil {
		return "", fmt.Errorf("Error expanding vars on the ingest pipeline path: %v", err)
	}

	return formatPipelineID(fs.mcfg.Module, fs.name, path, beatVersion), nil
}

// GetPipeline returns the JSON content of the Ingest Node pipeline that parses the logs.
func (fs *Fileset) GetPipeline(esVersion string) (pipelineID string, content map[string]interface{}, err error) {
	path, err := applyTemplate(fs.vars, fs.manifest.IngestPipeline, false)
	if err != nil {
		return "", nil, fmt.Errorf("Error expanding vars on the ingest pipeline path: %v", err)
	}

	strContents, err := ioutil.ReadFile(filepath.Join(fs.modulePath, fs.name, path))
	if err != nil {
		return "", nil, fmt.Errorf("Error reading pipeline file %s: %v", path, err)
	}

	vars, err := fs.turnOffElasticsearchVars(fs.vars, esVersion)
	if err != nil {
		return "", nil, err
	}

	jsonString, err := applyTemplate(vars, string(strContents), true)
	if err != nil {
		return "", nil, fmt.Errorf("Error interpreting the template of the ingest pipeline: %v", err)
	}

	err = json.Unmarshal([]byte(jsonString), &content)
	if err != nil {
		return "", nil, fmt.Errorf("Error JSON decoding the pipeline file: %s: %v", path, err)
	}
	return fs.pipelineID, content, nil
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
			ID:           fmt.Sprintf("filebeat-%s-%s-%s", fs.mcfg.Module, fs.name, ml.Name),
			JobPath:      filepath.Join(fs.modulePath, fs.name, ml.Job),
			DatafeedPath: filepath.Join(fs.modulePath, fs.name, ml.Datafeed),
			MinVersion:   ml.MinVersion,
		})
	}
	return mlConfigs
}
