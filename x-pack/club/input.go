package main

import (
	"fmt"
	"runtime/debug"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/go-concert/unison"
)

type inputLoader struct {
	log      *logp.Logger
	registry v2.Registry
}

type input struct {
	inputMeta  inputMeta
	streamMeta streamMeta
	configHash string
	useOutput  string
	runner     v2.Input
}

type streamMeta struct {
	ID      string `config:"id"`
	DataSet string `config:"data_stream.dataset"`
}

type inputMeta struct {
	ID        string                 `config:"id"`
	Name      string                 `config:"name"`
	Type      string                 `config:"type"`
	Meta      map[string]interface{} `config:"name"`
	Namespace string                 `config:"data_stream.namespace"`
}

type inputSettings struct {
	ID              string                 `config:"id"`
	Name            string                 `config:"name"`
	Type            string                 `config:"type"`
	Meta            map[string]interface{} `config:"name"`
	Namespace       string                 `config:"data_stream.namespace"`
	UseOutput       string                 `config:"use_output"`
	DefaultSettings *common.Config         `config:"default"`
	Streams         []*common.Config       `config:"streams"`
}

func newInputLoader(log *logp.Logger, registry v2.Registry) *inputLoader {
	return &inputLoader{log: log, registry: registry}
}

func (l *inputLoader) Init(group unison.Group, mode v2.Mode) error {
	return l.registry.Init(group, mode)
}

func (l *inputLoader) Configure(settings inputSettings) ([]*input, error) {
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
			configHash: streamConfig.Hash(),
			useOutput:  useOutput,
			runner:     runner,
		})
	}

	return inputs, nil
}

func (l *inputLoader) findInputPlugin(settings inputSettings, streamConfig *common.Config) (string, v2.Plugin, error) {
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

func (inp *input) Test(ctx v2.TestContext) error {
	return inp.runner.Test(ctx)
}

func (inp *input) Run(ctx v2.Context, pipeline beat.Pipeline) (err error) {
	ctx.Logger = inp.inputMeta.loggerWith(ctx.Logger)
	ctx.Logger = inp.streamMeta.loggerWith(ctx.Logger)

	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("input panic with: %+v\n%s", v, debug.Stack())
			ctx.Logger.Errorf("Input crashed with: %+v", err)
		}
	}()

	return inp.runner.Run(ctx, pipeline)
}

func (m *inputMeta) loggerWith(log *logp.Logger) *logp.Logger {
	log = log.With("type", m.Type)
	if m.ID != "" {
		log = log.With("input_id", m.ID)
	}
	if m.Name != "" {
		log = log.With("input_name", m.Name)
	}
	return log
}

func (m *streamMeta) loggerWith(log *logp.Logger) *logp.Logger {
	if m.ID != "" {
		log = log.With("input_id", m.ID)
	}
	return log
}
