package fileset

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/paths"
)

type ModuleRegistry struct {
	registry map[string]map[string]*Fileset // module -> fileset -> Fileset
}

// newModuleRegistry reads and loads the configured module into the registry.
func newModuleRegistry(modulesPath string,
	moduleConfigs []ModuleConfig,
	overrides *ModuleOverrides) (*ModuleRegistry, error) {

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

			if fcfg.Enabled != nil && (*fcfg.Enabled) == false {
				continue
			}

			fcfg, err = applyOverrides(fcfg, mcfg.Module, filesetName, overrides)
			if err != nil {
				return nil, fmt.Errorf("Error applying overrides on fileset %s/%s: %v", mcfg.Module, filesetName, err)
			}

			fileset, err := New(modulesPath, filesetName, &mcfg, fcfg)
			if err != nil {
				return nil, err
			}
			err = fileset.Read()
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
func NewModuleRegistry(moduleConfigs []*common.Config) (*ModuleRegistry, error) {
	modulesPath := paths.Resolve(paths.Home, "module")
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
	return newModuleRegistry(modulesPath, mcfgs, modulesOverrides)
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
