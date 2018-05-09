package hints

import (
	"fmt"
	"regexp"

	"github.com/elastic/beats/filebeat/fileset"
	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/builder"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	autodiscover.Registry.AddBuilder("hints", NewLogHints)
}

const (
	multiline    = "multiline"
	includeLines = "include_lines"
	excludeLines = "exclude_lines"
)

// validModuleNames to sanitize user input
var validModuleNames = regexp.MustCompile("[^a-zA-Z0-9]+")

type logHints struct {
	Key      string
	Config   *common.Config
	Registry *fileset.ModuleRegistry
}

// NewLogHints builds a log hints builder
func NewLogHints(cfg *common.Config) (autodiscover.Builder, error) {
	cfgwarn.Beta("The hints builder is beta")
	config := defaultConfig()
	err := cfg.Unpack(&config)

	if err != nil {
		return nil, fmt.Errorf("unable to unpack hints config due to error: %v", err)
	}

	moduleRegistry, err := fileset.NewModuleRegistry([]*common.Config{}, "", false)
	if err != nil {
		return nil, err
	}

	return &logHints{config.Key, config.Config, moduleRegistry}, nil
}

// Create config based on input hints in the bus event
func (l *logHints) CreateConfig(event bus.Event) []*common.Config {
	// Clone original config
	config, _ := common.NewConfigFrom(l.Config)
	host, _ := event["host"].(string)
	if host == "" {
		return []*common.Config{}
	}

	var hints common.MapStr
	hIface, ok := event["hints"]
	if ok {
		hints, _ = hIface.(common.MapStr)
	}

	if builder.IsNoOp(hints, l.Key) == true {
		return []*common.Config{config}
	}

	tempCfg := common.MapStr{}
	mline := l.getMultiline(hints)
	if len(mline) != 0 {
		tempCfg.Put(multiline, mline)
	}
	if ilines := l.getIncludeLines(hints); len(ilines) != 0 {
		tempCfg.Put(includeLines, ilines)
	}
	if elines := l.getExcludeLines(hints); len(elines) != 0 {
		tempCfg.Put(excludeLines, elines)
	}

	// Merge config template with the configs from the annotations
	if err := config.Merge(tempCfg); err != nil {
		logp.Debug("hints.builder", "config merge failed with error: %v", err)
		return []*common.Config{config}
	}

	module := l.getModule(hints)
	if module != "" {
		moduleConf := map[string]interface{}{
			"module": module,
		}

		filesets := l.getFilesets(hints, module)
		for fileset, conf := range filesets {
			filesetConf, _ := common.NewConfigFrom(config)
			filesetConf.SetString("containers.stream", -1, conf.Stream)

			moduleConf[fileset+".enabled"] = conf.Enabled
			moduleConf[fileset+".input"] = filesetConf

			logp.Debug("hints.builder", "generated config %+v", moduleConf)
		}
		config, _ = common.NewConfigFrom(moduleConf)
	}

	logp.Debug("hints.builder", "generated config %+v", config)

	// Apply information in event to the template to generate the final config
	return template.ApplyConfigTemplate(event, []*common.Config{config})
}

func (l *logHints) getMultiline(hints common.MapStr) common.MapStr {
	return builder.GetHintMapStr(hints, l.Key, multiline)
}

func (l *logHints) getIncludeLines(hints common.MapStr) []string {
	return builder.GetHintAsList(hints, l.Key, includeLines)
}

func (l *logHints) getExcludeLines(hints common.MapStr) []string {
	return builder.GetHintAsList(hints, l.Key, excludeLines)
}

func (l *logHints) getModule(hints common.MapStr) string {
	module := builder.GetHintString(hints, l.Key, "module")
	// for security, strip module name
	return validModuleNames.ReplaceAllString(module, "")
}

type filesetConfig struct {
	Enabled bool
	Stream  string
}

// Return a map containing filesets -> enabled & stream (stdout, stderr, all)
func (l *logHints) getFilesets(hints common.MapStr, module string) map[string]*filesetConfig {
	var configured bool
	filesets := make(map[string]*filesetConfig)

	moduleFilesets, err := l.Registry.ModuleFilesets(module)
	if err != nil {
		logp.Err("Error retrieving module filesets", err)
		return nil
	}

	for _, fileset := range moduleFilesets {
		filesets[fileset] = &filesetConfig{Enabled: false, Stream: "all"}
	}

	// If a single fileset is given, pass all streams to it
	fileset := builder.GetHintString(hints, l.Key, "fileset")
	if fileset != "" {
		if conf, ok := filesets[fileset]; ok {
			conf.Enabled = true
			configured = true
		}
	}

	// If fileset is defined per stream, return all of them
	for _, stream := range []string{"all", "stdout", "stderr"} {
		fileset := builder.GetHintString(hints, l.Key, "fileset."+stream)
		if fileset != "" {
			if conf, ok := filesets[fileset]; ok {
				conf.Enabled = true
				conf.Stream = stream
				configured = true
			}
		}
	}

	// No fileseat defined, return defaults for the module, all streams to all filesets
	if !configured {
		for _, conf := range filesets {
			conf.Enabled = true
		}
	}

	return filesets
}
