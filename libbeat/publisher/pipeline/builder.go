package pipeline

import (
	"errors"
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

var ErrNoReloadPipelineAlreadyBuilt = errors.New("cannot reload as Pipeline has been already built")

// Builder is an implementation of Builder pattern for building a Pipeline
type Builder struct {
	// Pipeline parameters
	beatInfo   beat.Info
	monitors   Monitors
	config     Config
	makeOutput outputFactory
	settings   Settings

	// Attributes for lazy loading of the pipeline
	buildmx       sync.Mutex
	pipelineBuilt bool
	pipeline      *Pipeline
	pipelineErr   error

	// Reloaders
	outputReloader          *outputReloader
	globalProcessorReloader *globalProcessorReloader

	processingFactory processing.SupportFactory
}

func NewPipelineBuilder(beatInfo beat.Info, monitors Monitors, config Config, outFactory outputFactory, settings Settings, processorsFactory processing.SupportFactory) *Builder {
	b := &Builder{
		beatInfo:          beatInfo,
		monitors:          monitors,
		config:            config,
		makeOutput:        outFactory,
		settings:          settings,
		processingFactory: processorsFactory,
	}

	b.outputReloader = &outputReloader{b}
	b.globalProcessorReloader = &globalProcessorReloader{b: b}

	return b
}

func (b *Builder) WithGlobalProcessors(processors processing.Supporter) error {
	b.settings.Processors = processors
	return nil
}

// build will materialize the Pipeline only once, after Pipeline has been built this is a no-op
func (b *Builder) build() {
	b.buildmx.Lock()
	defer b.buildmx.Unlock()
	log := b.monitors.Logger

	if b.pipelineBuilt {
		log.Debug("Pipeline already built, skipping..")
		return
	}

	log.Info("Creating Pipeline...")
	p, err := LoadWithSettings(b.beatInfo, b.monitors, b.config, b.makeOutput, b.settings)
	b.pipelineBuilt = true

	if err != nil {
		log.Errorf("Error creating pipeline: %s", err)
		b.pipelineErr = fmt.Errorf("instantiating Pipeline: %w", err)
		return
	}
	b.pipeline = p
	log.Info("Pipeline created successfully")
}

func (b *Builder) ConnectWith(config beat.ClientConfig) (beat.Client, error) {
	b.build()
	if b.pipelineErr != nil {
		return nil, b.pipelineErr
	}

	return b.pipeline.ConnectWith(config)
}

func (b *Builder) Connect() (beat.Client, error) {
	b.build()
	if b.pipelineErr != nil {
		return nil, b.pipelineErr
	}

	return b.pipeline.Connect()
}

func (b *Builder) OutputReloader() OutputReloader {
	return b.outputReloader
}

func (b *Builder) GlobalProcessorsReloader() *globalProcessorReloader {
	return b.globalProcessorReloader
}

type globalProcessorReloader struct {
	b *Builder
}

func (gpr *globalProcessorReloader) Reload(config *reload.ConfigWithMeta) error {
	builder := gpr.b
	builder.buildmx.Lock()
	defer builder.buildmx.Unlock()

	builder.monitors.Logger.Debugf("Reloading global processor with %s", config.Config)

	if builder.pipelineBuilt {
		// Too late as the pipeline is built already. We need to restart
		builder.monitors.Logger.Debug("Pipeline already instantiated. Returning ErrNoReloadPipelineAlreadyBuilt")
		return ErrNoReloadPipelineAlreadyBuilt
	}

	newProcessors, err := gpr.createProcessors(config.Config)
	if err != nil {
		return fmt.Errorf("creating new processors with config %s : %w", config.Config, err)
	}
	builder.WithGlobalProcessors(newProcessors)
	builder.monitors.Logger.Debugf("Reloading global processor complete", config.Config)
	return nil
}

func (gpr *globalProcessorReloader) createProcessors(rawProcessorConfig *conf.C) (processing.Supporter, error) {
	processingFactory := gpr.b.processingFactory
	if processingFactory == nil {
		processingFactory = processing.MakeDefaultBeatSupport(true)
	}
	return processingFactory(gpr.b.beatInfo, logp.L().Named("processors"), rawProcessorConfig)
}

type outputReloader struct {
	b *Builder
}

func (or *outputReloader) Reload(
	cfg *reload.ConfigWithMeta,
	factory func(outputs.Observer, conf.Namespace) (outputs.Group, error),
) error {
	or.b.build() // create the pipeline if needed
	if or.b.pipelineErr != nil {
		return or.b.pipelineErr
	}
	return or.b.pipeline.OutputReloader().Reload(cfg, factory)
}
