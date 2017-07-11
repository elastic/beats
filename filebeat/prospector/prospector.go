package prospector

import (
	"fmt"
	"sync"
	"time"

	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/filebeat/prospector/log"
	"github.com/elastic/beats/filebeat/prospector/redis"
	"github.com/elastic/beats/filebeat/prospector/stdin"
	"github.com/elastic/beats/filebeat/prospector/udp"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Prospector contains the prospector
type Prospector struct {
	config       prospectorConfig
	prospectorer Prospectorer
	done         chan struct{}
	wg           *sync.WaitGroup
	id           uint64
	Once         bool
	beatDone     chan struct{}
}

// Prospectorer is the interface common to all prospectors
type Prospectorer interface {
	Run()
	Stop()
	Wait()
}

// NewProspector instantiates a new prospector
func NewProspector(
	conf *common.Config,
	outlet channel.OutleterFactory,
	beatDone chan struct{},
	states []file.State,
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
	prospector.id, err = hashstructure.Hash(h, nil)
	if err != nil {
		return nil, err
	}

	err = prospector.initProspectorer(outlet, states, conf)
	if err != nil {
		return prospector, err
	}

	return prospector, nil
}

func (p *Prospector) initProspectorer(outlet channel.OutleterFactory, states []file.State, config *common.Config) error {

	var prospectorer Prospectorer
	var err error

	switch p.config.Type {
	case harvester.StdinType:
		prospectorer, err = stdin.NewProspector(config, outlet)
	case harvester.RedisType:
		prospectorer, err = redis.NewProspector(config, outlet)
	case harvester.LogType:
		prospectorer, err = log.NewProspector(config, states, outlet, p.done, p.beatDone)
	case harvester.UdpType:
		prospectorer, err = udp.NewProspector(config, outlet)
	default:
		return fmt.Errorf("invalid prospector type: %v. Change type", p.config.Type)
	}

	if err != nil {
		return err
	}

	p.prospectorer = prospectorer

	return nil
}

// Start starts the prospector
func (p *Prospector) Start() {
	p.wg.Add(1)
	logp.Info("Starting prospector of type: %v; id: %v ", p.config.Type, p.ID())

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

// ID returns prospector identifier
func (p *Prospector) ID() uint64 {
	return p.id
}

// Stop stops the prospector and with it all harvesters
func (p *Prospector) Stop() {
	// Stop scanning and wait for completion
	close(p.done)
	p.wg.Wait()
}

func (p *Prospector) stop() {
	logp.Info("Stopping Prospector: %v", p.ID())

	// In case of once, it will be waited until harvesters close itself
	if p.Once {
		p.prospectorer.Wait()
	} else {
		p.prospectorer.Stop()
	}
}
