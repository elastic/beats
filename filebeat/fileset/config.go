package fileset

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
	Prospector map[string]interface{} `config:"prospector"`
}

var defaultFilesetConfig = FilesetConfig{}
