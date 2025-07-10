package inputconcurrent

import (
	"fmt"
	"runtime/debug"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management/status"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
)

type concurrentInputManager struct {
	inputType string
	configure func(*conf.C) (Input, error)
}

type config struct {
	NumPipelineWorkers int    `config:"number_of_workers" validate:"positive,nonzero"`
	Host               string `config:"host"`
}

// ConfigureWith creates an InputManager that provides no extra logic and
// allows each input to fully control event collection and publishing in
// isolation. The function fn will be called for every input to be configured.
func New(fn func(*conf.C) (Input, error)) v2.InputManager {
	return &concurrentInputManager{configure: fn}
}

// Init is required to fulfil the input.InputManager interface.
// For the kafka input no special initialization is required.
func (*concurrentInputManager) Init(grp unison.Group) error { return nil }

// Create builds a new Input instance from the given configuration, or returns
// an error if the configuration is invalid.
func (manager *concurrentInputManager) Create(cfg *conf.C) (v2.Input, error) {
	wrapperCfg := config{NumPipelineWorkers: 1}
	if err := cfg.Unpack(&wrapperCfg); err != nil {
		return nil, err
	}

	inp, err := manager.configure(cfg)
	if err != nil {
		return nil, err
	}

	w := wrapper{
		inp:                inp,
		NumPipelineWorkers: wrapperCfg.NumPipelineWorkers,
		host:               wrapperCfg.Host,
		evtChan:            make(chan beat.Event),
	}

	return w, nil
}

type Input interface {
	Name() string
	Test(v2.TestContext) error
	InitMetrics(string, *logp.Logger) Metrics
	Run(v2.Context, chan<- beat.Event, Metrics) error
}

type Metrics interface {
	EventPublished(start time.Time)
	EventReceived(len int, timestamp time.Time)
}

type wrapper struct {
	inp                Input
	NumPipelineWorkers int
	evtChan            chan beat.Event
	host               string // used for metrics
}

// Name reports the input name.
//
// XXX: check if/how we can remove this method. Currently it is required for
// compatibility reasons with existing interfaces in libbeat, autodiscovery
// and filebeat.
func (w wrapper) Name() string { return w.inp.Name() }

// Test checks the configuration and runs additional checks if the Input can
// actually collect data for the given configuration (e.g. check if host/port or files are
// accessible).
func (w wrapper) Test(ctx v2.TestContext) error { return w.inp.Test(ctx) }

// Run starts the data collection. Run must return an error only if the
// error is fatal making it impossible for the input to recover.
func (w wrapper) Run(ctx v2.Context, pipeline beat.PipelineConnector) (err error) {
	logger := ctx.Logger.With("host", w.host)
	ctx.Logger = logger

	defer func() {
		if v := recover(); v != nil {
			if e, ok := v.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("TCP input panic with: %+v\n%s", v, debug.Stack())
			}
			logger.Errorw("TCP input panic", err)
		}
	}()

	logger.Infof("starting %s input", w.inp.Name())
	defer logger.Infof("%s input stopped", w.inp.Name())

	ctx.UpdateStatus(status.Starting, "")
	ctx.UpdateStatus(status.Configuring, "")

	m := w.inp.InitMetrics(ctx.ID, ctx.Logger)
	w.initWorkers(ctx, pipeline, m)
	w.inp.Run(ctx, w.evtChan, m)

	return nil
}

func (s wrapper) initWorkers(ctx v2.Context, pipeline beat.Pipeline, metrics Metrics) error {
	clients := []beat.Client{}
	for id := range s.NumPipelineWorkers {
		client, err := pipeline.ConnectWith(beat.ClientConfig{
			PublishMode: beat.DefaultGuarantees,
		})
		if err != nil {
			return fmt.Errorf("[worker %0d] cannot connect to publishing pipeline: %w", id, err)
		}

		clients = append(clients, client)
		go s.publishLoop(ctx, id, client, metrics)
	}

	// Close all clients when the input is closed
	go func() {
		select {
		case <-ctx.Cancelation.Done():
		}
		for _, c := range clients {
			c.Close()
		}
	}()

	return nil
}

func (s wrapper) publishLoop(ctx v2.Context, id int, publisher stateless.Publisher, metrics Metrics) {
	logger := ctx.Logger
	logger.Debugf("[Worker %d] starting publish loop", id)
	defer logger.Debugf("[Worker %d] finished publish loop", id)
	for {
		select {
		case <-ctx.Cancelation.Done():
			logger.Debugf("[Worker %d] Context cancelled, closing publish Loop", id)
			return
		case evt := <-s.evtChan:
			start := time.Now()
			publisher.Publish(evt)
			metrics.EventPublished(start)
		}
	}
}
