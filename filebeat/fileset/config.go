package fileset

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
)

// ModuleConfig contains the configuration file options for a module
type ModuleConfig struct {
	Module  string `config:"module"     validate:"required"`
	Enabled *bool  `config:"enabled"`

	// Filesets is inlined by code, see mcfgFromConfig
	Filesets map[string]*FilesetConfig
}

// FilesetConfig contains the configuration file options for a fileset
type FilesetConfig struct {
	Enabled    *bool                  `config:"enabled"`
	Var        map[string]interface{} `config:"var"`
	Input      map[string]interface{} `config:"input"`
	Prospector map[string]interface{} `config:"prospector"`
}

// NewFilesetConfig creates a new FilesetConfig from a common.Config.
func NewFilesetConfig(cfg *common.Config) (*FilesetConfig, error) {
	var fcfg FilesetConfig
	err := cfg.Unpack(&fcfg)
	if err != nil {
		return nil, fmt.Errorf("error unpacking configuration")
	}

	if len(fcfg.Prospector) > 0 {
		cfgwarn.Deprecate("7.0.0", "prospector is deprecated. Use `input` instead.")
		if len(fcfg.Input) > 0 {
			return nil, fmt.Errorf("error prospector and input are defined in the fileset, use only input")
		}
		fcfg.Input = fcfg.Prospector
	}
	return &fcfg, nil
}
