package pipeline

import (
	"fmt"
	"runtime/debug"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
)

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
