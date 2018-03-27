package hints

import (
	"fmt"

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

type logHints struct {
	Key    string
	Config []*common.Config
}

// NewLogHints builds a log hints builder
func NewLogHints(cfg *common.Config) (autodiscover.Builder, error) {
	cfgwarn.Beta("The hints builder is beta")
	config := defaultConfig()
	err := cfg.Unpack(&config)

	if err != nil {
		return nil, fmt.Errorf("unable to unpack hints config due to error: %v", err)
	}

	return &logHints{config.Key, config.Config}, nil
}

// Create config based on input hints in the bus event
func (l *logHints) CreateConfig(event bus.Event) []*common.Config {
	var config []*common.Config

	host, _ := event["host"].(string)
	if host == "" {
		return config
	}

	var hints common.MapStr
	hIface, ok := event["hints"]
	if ok {
		hints, _ = hIface.(common.MapStr)
	}

	if builder.IsNoOp(hints, l.Key) == true {
		return config
	}

	//TODO: Add module support

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
	for _, c := range l.Config {
		if err := c.Merge(tempCfg); err != nil {
			logp.Debug("hints.builder", "config merge failed with error: %v", err)
		} else {
			logp.Debug("hints.builder", "generated config %v", *c)
			config = append(config, c)
		}
	}

	// Apply information in event to the template to generate the final config
	config = template.ApplyConfigTemplate(event, config)
	return config
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
