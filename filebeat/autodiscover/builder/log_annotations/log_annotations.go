package log_annotations

import (
	"fmt"

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/builder"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	autodiscover.Registry.AddBuilder("log.annotations", NewLogAnnotations)
}

type logAnnotations struct {
	Key    string
	Config []*common.Config
}

// Construct a log annotations builder
func NewLogAnnotations(cfg *common.Config) (autodiscover.Builder, error) {
	config := defaultConfig()
	err := cfg.Unpack(&config)

	if err != nil {
		return nil, fmt.Errorf("unable to unpack log.annotations config due to error: %v", err)
	}

	return &logAnnotations{config.Key, config.Config}, nil
}

// Create config based on input hints in the bus event
func (l *logAnnotations) CreateConfig(event bus.Event) []*common.Config {
	var config []*common.Config

	host, _ := event["host"].(string)
	if host == "" {
		return config
	}

	var hints common.MapStr
	hIface, ok := event["hints"]
	if !ok {
		return config
	} else {
		hints, _ = hIface.(common.MapStr)
	}

	if builder.IsNoOp(hints, l.Key) == true {
		return config
	}

	//TODO: Add module support

	tempCfg := common.MapStr{}
	multiline := l.getMultiline(hints)

	for k, v := range multiline {
		tempCfg.Put(k, v)
	}
	if includeLines := l.getIncludeLines(hints); len(includeLines) != 0 {
		tempCfg.Put("include_lines", includeLines)
	}
	if excludeLines := l.getExcludeLines(hints); len(excludeLines) != 0 {
		tempCfg.Put("exclude_lines", excludeLines)
	}

	// Merge config template with the configs from the annotations
	for _, c := range l.Config {
		if err := c.Merge(tempCfg); err != nil {
			logp.Debug("log.annotations", "config merge failed with error: %v", err)
		} else {
			cfg := common.MapStr{}
			c.Unpack(cfg)
			logp.Debug("log.annotations", "generated config %v", cfg.String())
			config = append(config, c)
		}
	}

	// Apply information in event to the template to generate the final config
	config = template.ApplyConfigTemplate(event, config)
	return config
}

func (l *logAnnotations) getMultiline(hints common.MapStr) common.MapStr {
	return builder.GetHintMapStr(hints, l.Key, "multiline")
}

func (l *logAnnotations) getIncludeLines(hints common.MapStr) []string {
	return builder.GetHintAsList(hints, l.Key, "include_lines")
}

func (l *logAnnotations) getExcludeLines(hints common.MapStr) []string {
	return builder.GetHintAsList(hints, l.Key, "exclude_lines")
}
