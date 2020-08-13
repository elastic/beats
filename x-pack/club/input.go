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
	"github.com/urso/sderr"
)

type inputLoader struct {
	log      *logp.Logger
	registry v2.Registry
}

type input struct {
	streams   []stream
	meta      inputMeta
	useOutput string
}

type inputMeta struct {
	ID        string                 `config:"id"`
	Name      string                 `config:"name"`
	Type      string                 `config:"type"`
	Meta      map[string]interface{} `config:"name"`
	Namespace string                 `config:"dataset.namespace"`
}

type stream struct {
	meta   streamMeta
	runner v2.Input
}

type streamMeta struct {
	ID string `config:"id"`
}

type inputSettings struct {
	ID              string                 `config:"id"`
	Name            string                 `config:"name"`
	Type            string                 `config:"type"`
	Meta            map[string]interface{} `config:"name"`
	Namespace       string                 `config:"dataset.namespace"`
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

func (l *inputLoader) Configure(settings inputSettings) (*input, error) {
	p, exists := l.registry.Find(settings.Type)
	if !exists {
		return nil, &v2.LoadError{Name: settings.Type, Reason: v2.ErrUnknownInput}
	}

	defaults := settings.DefaultSettings
	if defaults == nil {
		defaults = common.NewConfig()
	}

	streams := make([]stream, len(settings.Streams))
	for i, streamConfig := range settings.Streams {
		inputConfig := defaults.Clone()
		inputConfig.Merge(streamConfig)

		runner, err := p.Manager.Create(inputConfig)
		if err != nil {
			return nil, err
		}
		streams[i] = stream{
			runner: runner,
		}
	}

	useOutput := settings.UseOutput
	if useOutput == "" {
		useOutput = "default"
	}

	meta := inputMeta{
		ID:        settings.ID,
		Name:      settings.Name,
		Type:      settings.Type,
		Meta:      settings.Meta,
		Namespace: settings.Namespace,
	}

	log := meta.logWith(l.log)
	switch p.Stability {
	case feature.Experimental:
		log.Warnf("EXPERIMENTAL: The %v input is experimental", settings.Type)
	case feature.Beta:
		log.Warnf("BETA: The %v input is beta", settings.Type)
	}
	if p.Deprecated {
		log.Warnf("DEPRECATED: The %v input is deprecated", settings.Type)
	}

	return &input{meta: meta, streams: streams, useOutput: useOutput}, nil
}

func (inp *input) Test(ctx v2.TestContext) error {
	var grp unison.MultiErrGroup
	for _, stream := range inp.streams {
		stream := stream
		grp.Go(func() (err error) {
			return stream.runner.Test(ctx)
		})
	}
	return sderr.WrapAll(grp.Wait(), "input tests failed")
}

func (inp *input) Run(ctx v2.Context, pipeline beat.Pipeline) error {
	ctx.Logger = inp.meta.logWith(ctx.Logger)

	// We wait for all inputs to complete or fail individually. The input allows
	// existing streams to continue, even if a subset of streams have failed.
	var grp unison.MultiErrGroup
	for _, stream := range inp.streams {
		stream := stream
		grp.Go(func() (err error) {
			streamCtx := ctx
			streamCtx.Logger = stream.meta.loggerWith(ctx.Logger)

			return inp.runStream(streamCtx, pipeline, stream)
		})
	}

	return sderr.WrapAll(grp.Wait(), "input failures")
}

func (inp *input) runStream(ctx v2.Context, pipeline beat.Pipeline, stream stream) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("input panic with: %+v\n%s", v, debug.Stack())
			ctx.Logger.Errorf("Input crashed with: %+v", err)
		}
	}()

	return stream.runner.Run(ctx, pipeline)
}

func (m *inputMeta) logWith(log *logp.Logger) *logp.Logger {
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
