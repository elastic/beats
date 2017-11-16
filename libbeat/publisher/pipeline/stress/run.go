package stress

import (
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher/pipeline"
)

type config struct {
	Generate generateConfig         `config:"generate"`
	Pipeline pipeline.Config        `config:"pipeline"`
	Output   common.ConfigNamespace `config:"output"`
}

var defaultConfig = config{
	Generate: defaultGenerateConfig,
}

func RunTests(
	info beat.Info,
	duration time.Duration,
	cfg *common.Config,
	errors func(err error),
) error {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return err
	}

	// reg := monitoring.NewRegistry()
	pipeline, err := pipeline.Load(info, nil, config.Pipeline, config.Output)
	if err != nil {
		return err
	}
	defer func() {
		logp.Info("Stop pipeline")
		pipeline.Close()
		logp.Info("pipeline closed")
	}()

	cs := newCloseSignaler()

	// waitGroup for active generators
	var genWG sync.WaitGroup
	defer genWG.Wait() // block shutdown until all generators have quit

	for i := 0; i < config.Generate.Worker; i++ {
		i := i
		withWG(&genWG, func() {
			err := generate(cs, pipeline, config.Generate, i, errors)
			if err != nil {
				logp.Err("Generator failed with: %v", err)
			}
		})
	}

	if duration > 0 {
		// Note: don't care about the go-routine leaking (for now)
		go func() {
			time.Sleep(duration)
			cs.Close()
		}()
	}

	return nil
}

func withWG(wg *sync.WaitGroup, fn func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		fn()
	}()
}
