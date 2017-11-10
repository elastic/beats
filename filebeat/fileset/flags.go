package fileset

import (
	"flag"
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

// Modules related command line flags.
var (
	modulesFlag     = flag.String("modules", "", "List of enabled modules (comma separated)")
	moduleOverrides = common.SettingFlag(nil, "M", "Module configuration overwrite")
)

type ModuleOverrides map[string]map[string]*common.Config // module -> fileset -> Config

// Get returns an array of configuration overrides that should be merged in order.
func (mo *ModuleOverrides) Get(module, fileset string) []*common.Config {
	ret := []*common.Config{}

	moduleWildcard := (*mo)["*"]["*"]
	if moduleWildcard != nil {
		ret = append(ret, moduleWildcard)
	}

	filesetWildcard := (*mo)[module]["*"]
	if filesetWildcard != nil {
		ret = append(ret, filesetWildcard)
	}

	cfg := (*mo)[module][fileset]
	if cfg != nil {
		ret = append(ret, cfg)
	}

	return ret
}

func getModulesCLIConfig() ([]string, *ModuleOverrides, error) {
	modulesList := []string{}
	if modulesFlag != nil {
		modulesList = strings.Split(*modulesFlag, ",")
	}

	if moduleOverrides == nil {
		return modulesList, nil, nil
	}

	var overrides ModuleOverrides
	err := moduleOverrides.Unpack(&overrides)
	if err != nil {
		return []string{}, nil, fmt.Errorf("-M flags must be prefixed by the module and fileset: %v", err)
	}

	return modulesList, &overrides, nil
}
