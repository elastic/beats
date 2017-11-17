package prospector

import (
	"fmt"
	"sync"
	"time"

	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Prospectorer is the interface common to all prospectors
type Prospectorer interface {
	Run()
	Stop()
	Wait()
}

// Prospector contains the prospector
type Prospector struct {
	config       prospectorConfig
	prospectorer Prospectorer
	done         chan struct{}
	wg           *sync.WaitGroup
	ID           uint64
	Once         bool
	beatDone     chan struct{}
}

// NewProspector instantiates a new prospector
func New(
	conf *common.Config,
	outlet channel.Factory,
	beatDone chan struct{},
	states []file.State,
	dynFields *common.MapStrPointer,
) (*Prospector, error) {
	prospector := &Prospector{
		config:   defaultConfig,
		wg:       &sync.WaitGroup{},
		done:     make(chan struct{}),
		Once:     false,
		beatDone: beatDone,
	}

	var err error
	if err = conf.Unpack(&prospector.config); err != nil {
		return nil, err
	}

	var h map[string]interface{}
	conf.Unpack(&h)
	prospector.ID, err = hashstructure.Hash(h, nil)
	if err != nil {
		return nil, err
	}

	var f Factory
	f, err = GetFactory(prospector.config.Type)
	if err != nil {
		return prospector, err
	}

	context := Context{
		States:        states,
		Done:          prospector.done,
		BeatDone:      prospector.beatDone,
		DynamicFields: dynFields,
	}
	var prospectorer Prospectorer
	prospectorer, err = f(conf, outlet, context)
	if err != nil {
		return prospector, err
	}
	prospector.prospectorer = prospectorer

	return prospector, nil
}

// Start starts the prospector
func (p *Prospector) Start() {
	p.wg.Add(1)
	logp.Info("Starting prospector of type: %v; ID: %d ", p.config.Type, p.ID)

	onceWg := sync.WaitGroup{}
	if p.Once {
		// Make sure start is only completed when Run did a complete first scan
		defer onceWg.Wait()
	}

	onceWg.Add(1)
	// Add waitgroup to make sure prospectors finished
	go func() {
		defer func() {
			onceWg.Done()
			p.stop()
			p.wg.Done()
		}()

		p.Run()
	}()
}

// Run starts scanning through all the file paths and fetch the related files. Start a harvester for each file
func (p *Prospector) Run() {
	// Initial prospector run
	p.prospectorer.Run()

	// Shuts down after the first complete run of all prospectors
	if p.Once {
		return
	}

	for {
		select {
		case <-p.done:
			logp.Info("Prospector ticker stopped")
			return
		case <-time.After(p.config.ScanFrequency):
			logp.Debug("prospector", "Run prospector")
			p.prospectorer.Run()
		}
	}
}

// Stop stops the prospector and with it all harvesters
func (p *Prospector) Stop() {
	// Stop scanning and wait for completion
	close(p.done)
	p.wg.Wait()
}

func (p *Prospector) stop() {
	logp.Info("Stopping Prospector: %d", p.ID)

	// In case of once, it will be waited until harvesters close itself
	if p.Once {
		p.prospectorer.Wait()
	} else {
		p.prospectorer.Stop()
	}
}

func (p *Prospector) String() string {
	return fmt.Sprintf("prospector [type=%s, ID=%d]", p.config.Type, p.ID)
}
