package pipeline

import (
	"fmt"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/mitchellh/hashstructure"
)

type inputLoader struct {
	log      *logp.Logger
	registry v2.Registry
}

func newInputLoader(log *logp.Logger, registry v2.Registry) *inputLoader {
	return &inputLoader{log: log, registry: registry}
}

func (l *inputLoader) Configure(settings InputSettings) ([]*input, error) {
	defaults := settings.DefaultSettings
	if defaults == nil {
		defaults = common.NewConfig()
	}

	inputMeta := inputMeta{
		ID:        settings.ID,
		Name:      settings.Name,
		Type:      settings.Type,
		Meta:      settings.Meta,
		Namespace: settings.Namespace,
	}
	log := inputMeta.loggerWith(l.log)

	useOutput := settings.UseOutput
	if useOutput == "" {
		useOutput = "default"
	}

	inputs := make([]*input, 0, len(settings.Streams))
	for _, cfg := range settings.Streams {
		streamConfig := defaults.Clone()
		streamConfig.Merge(cfg)

		var streamMeta streamMeta
		if err := streamConfig.Unpack(&streamMeta); err != nil {
			return nil, err
		}

		name, plugin, err := l.findInputPlugin(settings, streamConfig)
		if err != nil {
			return nil, err
		}

		switch plugin.Stability {
		case feature.Experimental:
			log.Warnf("EXPERIMENTAL: The %v input is experimental", name)
		case feature.Beta:
			log.Warnf("BETA: The %v input is beta", name)
		}
		if plugin.Deprecated {
			log.Warnf("DEPRECATED: The %v input is deprecated", name)
		}

		runner, err := plugin.Manager.Create(streamConfig)
		if err != nil {
			return nil, err
		}

		inputs = append(inputs, &input{
			inputMeta:  inputMeta,
			streamMeta: streamMeta,
			configHash: inputConfigHash(inputMeta, streamMeta, useOutput, streamConfig),
			useOutput:  useOutput,
			runner:     runner,
		})
	}

	return inputs, nil
}

func (l *inputLoader) findInputPlugin(settings InputSettings, streamConfig *common.Config) (string, v2.Plugin, error) {
	streamSettings := struct {
		DataSet string `config:"data_stream.dataset"`
		Type    string `config:"type"`
	}{}

	if err := streamConfig.Unpack(&streamSettings); err != nil {
		return "", v2.Plugin{}, err
	}

	if t := streamSettings.Type; t != "" {
		p, exists := l.registry.Find(t)
		if !exists {
			return "", v2.Plugin{}, &v2.LoadError{Name: t, Reason: v2.ErrUnknownInput}
		}
		return t, p, nil
	}

	//input type names can follow multiple patterns. Lets precompute allowed names
	// first. The name order in the array specifies the priority of a given naming scheme.
	var names []string
	if t := streamSettings.DataSet; t != "" {
		names = append(names, t)
		if it := settings.Type; it != "" {
			names = append(names, fmt.Sprintf("%v.%v", it, t))
		}
	}
	if it := settings.Type; it != "" {
		names = append(names, it)
	}

	if len(names) == 0 {
		return "", v2.Plugin{}, &v2.LoadError{Message: "no input type configured", Reason: v2.ErrUnknownInput}
	}

	for _, name := range names {
		if p, exists := l.registry.Find(name); exists {
			return name, p, nil
		}
	}

	return "", v2.Plugin{}, &v2.LoadError{Name: names[0], Reason: v2.ErrUnknownInput}
}

func inputConfigHash(im inputMeta, sm streamMeta, out string, cfg *common.Config) string {
	metaHashData := struct {
		UseOutput  string
		InputMeta  inputMeta
		StreamNeta streamMeta
	}{out, im, sm}
	metaHash, _ := hashstructure.Hash(&metaHashData, nil)
	return fmt.Sprintf("%x-%v", metaHash, cfg.Hash())
}
