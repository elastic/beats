package fileset

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	mlimporter "github.com/elastic/beats/libbeat/ml-importer"
	"github.com/elastic/beats/libbeat/paths"
)

type ModuleRegistry struct {
	registry map[string]map[string]*Fileset // module -> fileset -> Fileset
}

// newModuleRegistry reads and loads the configured module into the registry.
func newModuleRegistry(modulesPath string,
	moduleConfigs []ModuleConfig,
	overrides *ModuleOverrides,
	beatVersion string) (*ModuleRegistry, error) {

	var reg ModuleRegistry
	reg.registry = map[string]map[string]*Fileset{}

	for _, mcfg := range moduleConfigs {
		if mcfg.Enabled != nil && (*mcfg.Enabled) == false {
			continue
		}

		reg.registry[mcfg.Module] = map[string]*Fileset{}
		moduleFilesets, err := getModuleFilesets(modulesPath, mcfg.Module)
		if err != nil {
			return nil, fmt.Errorf("Error getting filesets for module %s: %v", mcfg.Module, err)
		}

		for _, filesetName := range moduleFilesets {
			fcfg, exists := mcfg.Filesets[filesetName]
			if !exists {
				fcfg = &defaultFilesetConfig
			}

			fcfg, err = applyOverrides(fcfg, mcfg.Module, filesetName, overrides)
			if err != nil {
				return nil, fmt.Errorf("Error applying overrides on fileset %s/%s: %v", mcfg.Module, filesetName, err)
			}

			if fcfg.Enabled != nil && (*fcfg.Enabled) == false {
				continue
			}

			fileset, err := New(modulesPath, filesetName, &mcfg, fcfg)
			if err != nil {
				return nil, err
			}
			err = fileset.Read(beatVersion)
			if err != nil {
				return nil, fmt.Errorf("Error reading fileset %s/%s: %v", mcfg.Module, filesetName, err)
			}
			reg.registry[mcfg.Module][filesetName] = fileset
		}

		// check that no extra filesets are configured
		for filesetName, fcfg := range mcfg.Filesets {
			if fcfg.Enabled != nil && (*fcfg.Enabled) == false {
				continue
			}
			found := false
			for _, name := range moduleFilesets {
				if filesetName == name {
					found = true
				}
			}
			if !found {
				return nil, fmt.Errorf("Fileset %s/%s is configured but doesn't exist", mcfg.Module, filesetName)
			}
		}
	}

	return &reg, nil
}

// NewModuleRegistry reads and loads the configured module into the registry.
func NewModuleRegistry(moduleConfigs []*common.Config, beatVersion string) (*ModuleRegistry, error) {
	modulesPath := paths.Resolve(paths.Home, "module")

	stat, err := os.Stat(modulesPath)
	if err != nil || !stat.IsDir() {
		logp.Err("Not loading modules. Module directory not found: %s", modulesPath)
		return &ModuleRegistry{}, nil // empty registry, no error
	}

	modulesCLIList, modulesOverrides, err := getModulesCLIConfig()
	if err != nil {
		return nil, err
	}
	mcfgs := []ModuleConfig{}
	for _, moduleConfig := range moduleConfigs {
		mcfg, err := mcfgFromConfig(moduleConfig)
		if err != nil {
			return nil, fmt.Errorf("Error unpacking module config: %v", err)
		}
		mcfgs = append(mcfgs, *mcfg)
	}
	mcfgs, err = appendWithoutDuplicates(mcfgs, modulesCLIList)
	if err != nil {
		return nil, err
	}
	return newModuleRegistry(modulesPath, mcfgs, modulesOverrides, beatVersion)
}

func mcfgFromConfig(cfg *common.Config) (*ModuleConfig, error) {
	var mcfg ModuleConfig

	err := cfg.Unpack(&mcfg)
	if err != nil {
		return nil, err
	}

	var dict map[string]interface{}

	err = cfg.Unpack(&dict)
	if err != nil {
		return nil, fmt.Errorf("Error unpacking module %s in a dict: %v", mcfg.Module, err)
	}

	mcfg.Filesets = map[string]*FilesetConfig{}
	for name, filesetConfig := range dict {
		if name == "module" || name == "enabled" {
			continue
		}

		var fcfg FilesetConfig
		tmpCfg, err := common.NewConfigFrom(filesetConfig)
		if err != nil {
			return nil, fmt.Errorf("Error creating config from fileset %s/%s: %v", mcfg.Module, name, err)
		}
		err = tmpCfg.Unpack(&fcfg)
		if err != nil {
			return nil, fmt.Errorf("Error unpacking fileset %s/%s: %v", mcfg.Module, name, err)
		}
		mcfg.Filesets[name] = &fcfg

	}

	return &mcfg, nil
}

func getModuleFilesets(modulePath, module string) ([]string, error) {
	fileInfos, err := ioutil.ReadDir(filepath.Join(modulePath, module))
	if err != nil {
		return []string{}, err
	}

	filesets := []string{}
	for _, fi := range fileInfos {
		if fi.IsDir() {
			// check also that the `manifest.yml` file exists
			_, err = os.Stat(filepath.Join(modulePath, module, fi.Name(), "manifest.yml"))
			if err == nil {
				filesets = append(filesets, fi.Name())
			}
		}
	}

	return filesets, nil
}

func applyOverrides(fcfg *FilesetConfig,
	module, fileset string,
	overrides *ModuleOverrides) (*FilesetConfig, error) {

	if overrides == nil {
		return fcfg, nil
	}

	overridesConfigs := overrides.Get(module, fileset)
	if len(overridesConfigs) == 0 {
		return fcfg, nil
	}

	config, err := common.NewConfigFrom(fcfg)
	if err != nil {
		return nil, fmt.Errorf("Error creating vars config object: %v", err)
	}

	toMerge := []*common.Config{config}
	toMerge = append(toMerge, overridesConfigs...)

	resultConfig, err := common.MergeConfigs(toMerge...)
	if err != nil {
		return nil, fmt.Errorf("Error merging configs: %v", err)
	}

	var res FilesetConfig
	err = resultConfig.Unpack(&res)
	if err != nil {
		return nil, fmt.Errorf("Error unpacking configs: %v", err)
	}

	return &res, nil
}

// appendWithoutDuplicates appends basic module configuration for each module in the
// modules list, unless the same module is not already loaded.
func appendWithoutDuplicates(moduleConfigs []ModuleConfig, modules []string) ([]ModuleConfig, error) {
	if len(modules) == 0 {
		return moduleConfigs, nil
	}

	// built a dictionary with the configured modules
	modulesMap := map[string]bool{}
	for _, mcfg := range moduleConfigs {
		if mcfg.Enabled != nil && (*mcfg.Enabled) == false {
			continue
		}
		modulesMap[mcfg.Module] = true
	}

	// add the non duplicates to the list
	for _, module := range modules {
		if _, exists := modulesMap[module]; !exists {
			moduleConfigs = append(moduleConfigs, ModuleConfig{Module: module})
		}
	}
	return moduleConfigs, nil
}

func (reg *ModuleRegistry) GetProspectorConfigs() ([]*common.Config, error) {
	result := []*common.Config{}
	for module, filesets := range reg.registry {
		for name, fileset := range filesets {
			fcfg, err := fileset.getProspectorConfig()
			if err != nil {
				return result, fmt.Errorf("Error getting config for fielset %s/%s: %v",
					module, name, err)
			}
			result = append(result, fcfg)
		}
	}
	return result, nil
}

// PipelineLoader factory builds and returns a PipelineLoader
type PipelineLoaderFactory func() (PipelineLoader, error)

// PipelineLoader is a subset of the Elasticsearch client API capable of loading
// the pipelines.
type PipelineLoader interface {
	LoadJSON(path string, json map[string]interface{}) ([]byte, error)
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
}

// LoadPipelines loads the pipelines for each configured fileset.
func (reg *ModuleRegistry) LoadPipelines(esClient PipelineLoader) error {
	for module, filesets := range reg.registry {
		for name, fileset := range filesets {
			// check that all the required Ingest Node plugins are available
			requiredProcessors := fileset.GetRequiredProcessors()
			logp.Debug("modules", "Required processors: %s", requiredProcessors)
			if len(requiredProcessors) > 0 {
				err := checkAvailableProcessors(esClient, requiredProcessors)
				if err != nil {
					return fmt.Errorf("Error loading pipeline for fileset %s/%s: %v", module, name, err)
				}
			}

			pipelineID, content, err := fileset.GetPipeline()
			if err != nil {
				return fmt.Errorf("Error getting pipeline for fileset %s/%s: %v", module, name, err)
			}
			err = loadPipeline(esClient, pipelineID, content)
			if err != nil {
				return fmt.Errorf("Error loading pipeline for fileset %s/%s: %v", module, name, err)
			}
		}
	}
	return nil
}

// checkAvailableProcessors calls the /_nodes/ingest API and verifies that all processors listed
// in the requiredProcessors list are available in Elasticsearch. Returns nil if all required
// processors are available.
func checkAvailableProcessors(esClient PipelineLoader, requiredProcessors []ProcessorRequirement) error {

	var response struct {
		Nodes map[string]struct {
			Ingest struct {
				Processors []struct {
					Type string `json:"type"`
				} `json:"processors"`
			} `json:"ingest"`
		} `json:"nodes"`
	}
	status, body, err := esClient.Request("GET", "/_nodes/ingest", "", nil, nil)
	if err != nil {
		return fmt.Errorf("Error querying _nodes/ingest: %v", err)
	}
	if status > 299 {
		return fmt.Errorf("Error querying _nodes/ingest. Status: %d. Response body: %s", status, body)
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return fmt.Errorf("Error unmarshaling json when querying _nodes/ingest. Body: %s", body)
	}

	missing := []ProcessorRequirement{}
	for _, requiredProcessor := range requiredProcessors {
		for _, node := range response.Nodes {
			available := false
			for _, availableProcessor := range node.Ingest.Processors {
				if requiredProcessor.Name == availableProcessor.Type {
					available = true
					break
				}
			}
			if !available {
				missing = append(missing, requiredProcessor)
				break
			}
		}
	}

	if len(missing) > 0 {
		missingPlugins := []string{}
		for _, proc := range missing {
			missingPlugins = append(missingPlugins, proc.Plugin)
		}
		errorMsg := fmt.Sprintf("This module requires the following Elasticsearch plugins: %s. "+
			"You can install them by running the following commands on all the Elasticsearch nodes:",
			strings.Join(missingPlugins, ", "))
		for _, plugin := range missingPlugins {
			errorMsg += fmt.Sprintf("\n    sudo bin/elasticsearch-plugin install %s", plugin)
		}
		return errors.New(errorMsg)
	}

	return nil
}

func loadPipeline(esClient PipelineLoader, pipelineID string, content map[string]interface{}) error {
	path := "/_ingest/pipeline/" + pipelineID
	status, _, _ := esClient.Request("GET", path, "", nil, nil)
	if status == 200 {
		logp.Debug("modules", "Pipeline %s already loaded", pipelineID)
		return nil
	}
	body, err := esClient.LoadJSON(path, content)
	if err != nil {
		return interpretError(err, body)
	}
	logp.Info("Elasticsearch pipeline with ID '%s' loaded", pipelineID)
	return nil
}

func interpretError(initialErr error, body []byte) error {
	var response struct {
		Error struct {
			RootCause []struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
				Header struct {
					ProcessorType string `json:"processor_type"`
				} `json:"header"`
				Index string `json:"index"`
			} `json:"root_cause"`
		} `json:"error"`
	}
	err := json.Unmarshal(body, &response)
	if err != nil {
		// this might be ES < 2.0. Do a best effort to check for ES 1.x
		var response1x struct {
			Error string `json:"error"`
		}
		err1x := json.Unmarshal(body, &response1x)
		if err1x == nil && response1x.Error != "" {
			return fmt.Errorf("The Filebeat modules require Elasticsearch >= 5.0. "+
				"This is the response I got from Elasticsearch: %s", body)
		}

		return fmt.Errorf("couldn't load pipeline: %v. Additionally, error decoding response body: %s",
			initialErr, body)
	}

	// missing plugins?
	if len(response.Error.RootCause) > 0 &&
		response.Error.RootCause[0].Type == "parse_exception" &&
		strings.HasPrefix(response.Error.RootCause[0].Reason, "No processor type exists with name") &&
		response.Error.RootCause[0].Header.ProcessorType != "" {

		plugins := map[string]string{
			"geoip":      "ingest-geoip",
			"user_agent": "ingest-user-agent",
		}
		plugin, ok := plugins[response.Error.RootCause[0].Header.ProcessorType]
		if !ok {
			return fmt.Errorf("This module requires an Elasticsearch plugin that provides the %s processor. "+
				"Please visit the Elasticsearch documentation for instructions on how to install this plugin. "+
				"Response body: %s", response.Error.RootCause[0].Header.ProcessorType, body)
		}

		return fmt.Errorf("This module requires the %s plugin to be installed in Elasticsearch. "+
			"You can install it using the following command in the Elasticsearch home directory:\n"+
			"    sudo bin/elasticsearch-plugin install %s", plugin, plugin)
	}

	// older ES version?
	if len(response.Error.RootCause) > 0 &&
		response.Error.RootCause[0].Type == "invalid_index_name_exception" &&
		response.Error.RootCause[0].Index == "_ingest" {

		return fmt.Errorf("The Ingest Node functionality seems to be missing from Elasticsearch. "+
			"The Filebeat modules require Elasticsearch >= 5.0. "+
			"This is the response I got from Elasticsearch: %s", body)
	}

	return fmt.Errorf("couldn't load pipeline: %v. Response body: %s", initialErr, body)
}

// LoadML loads the machine-learning configurations into Elasticsearch, if Xpack is avaiable
func (reg *ModuleRegistry) LoadML(esClient PipelineLoader) error {
	haveXpack, err := mlimporter.HaveXpackML(esClient)
	if err != nil {
		return errors.Errorf("Error checking if xpack is available: %v", err)
	}
	if !haveXpack {
		logp.Warn("Xpack Machine Learning is not enabled")
		return nil
	}

	for module, filesets := range reg.registry {
		for name, fileset := range filesets {
			for _, mlConfig := range fileset.GetMLConfigs() {
				err = mlimporter.ImportMachineLearningJob(esClient, &mlConfig)
				if err != nil {
					return errors.Errorf("Error loading ML config from %s/%s: %v", module, name, err)
				}
			}
		}
	}

	return nil
}

func (reg *ModuleRegistry) Empty() bool {
	count := 0
	for _, filesets := range reg.registry {
		count += len(filesets)
	}
	return count == 0
}
