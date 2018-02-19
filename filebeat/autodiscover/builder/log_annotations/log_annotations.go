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
	Prefix string
	Config []*common.Config
}

func NewLogAnnotations(cfg *common.Config) (autodiscover.Builder, error) {
	config := defaultConfig()
	err := cfg.Unpack(&config)

	if err != nil {
		return nil, fmt.Errorf("unable to unpack log.annotations config due to error: %v", err)
	}

	return &logAnnotations{config.Prefix, config.Config}, nil
}

func (l *logAnnotations) CreateConfig(event bus.Event) []*common.Config {
	var config []*common.Config

	host, _ := event["host"].(string)
	if host == "" {
		return config
	}

	annotations, ok := event["annotations"].(map[string]string)
	if !ok {
		return config
	}

	container, ok := event["container"].(common.MapStr)
	if !ok {
		return config
	}
	id := builder.GetContainerID(container)

	if builder.IsContainerNoOp(annotations, l.Prefix, id) == true {
		return config
	}

	//TODO: Add module support

	tempCfg := common.MapStr{}
	multiline := l.getMultiline(annotations, id)

	for k, v := range multiline {
		tempCfg.Put(k, v)
	}
	if includeLines := l.getIncludeLines(annotations, id); len(includeLines) != 0 {
		tempCfg.Put("include_lines", includeLines)
	}
	if excludeLines := l.getExcludeLines(annotations, id); len(excludeLines) != 0 {
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

func (l *logAnnotations) getMultiline(annotations map[string]string, container string) map[string]string {
	return builder.GetContainerAnnotationsWithPrefix(annotations, l.Prefix, container, "multiline")
}

func (l *logAnnotations) getIncludeLines(annotations map[string]string, container string) []string {
	return builder.GetContainerAnnotationsAsList(annotations, l.Prefix, container, "include_lines")
}

func (l *logAnnotations) getExcludeLines(annotations map[string]string, container string) []string {
	return builder.GetContainerAnnotationsAsList(annotations, l.Prefix, container, "exclude_lines")
}
