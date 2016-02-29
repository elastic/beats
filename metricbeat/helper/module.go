package helper

import (
	"sync"
	"time"

	"fmt"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"
)

// Module specifics. This must be defined by each module
type Module struct {
	name string

	// Moduler implementation
	moduler Moduler

	// Module config
	config ModuleConfig

	// List of all metricsets in this module. Use to keep track of metricsets
	metricSets map[string]*MetricSet

	// MetricSet waitgroup
	wg sync.WaitGroup

	done chan struct{}
}

// NewModule creates a new module
func NewModule(config ModuleConfig, moduler Moduler) *Module {
	return &Module{
		name:       config.Module,
		config:     config,
		moduler:    moduler,
		metricSets: map[string]*MetricSet{},
		wg:         sync.WaitGroup{},
		done:       make(chan struct{}),
	}
}

// Starts the given module
func (m *Module) Start(b *beat.Beat) {

	defer logp.Recover(fmt.Sprintf("Module %s paniced and stopped running.", m.name))

	if !m.config.Enabled {
		logp.Debug("helper", "Not starting module %s as not enabled.", m.name)
		return
	}

	logp.Info("Setup moduler: %s", m.name)
	err := m.moduler.Setup()
	if err != nil {
		logp.Err("Error setting up module: %s. Not starting metricsets for this module.", err)
		return
	}

	// Setup - Create metricSets for the module
	for _, metricsetName := range m.config.MetricSets {
		metricSeter := Registry.MetricSeters[m.name][metricsetName]
		// Setup
		err := metricSeter.Setup()
		if err != nil {
			logp.Err("Error happening during metricseter setup: %s", err)
		}

		metricSet := NewMetricSet(metricsetName, metricSeter, m.config)
		m.metricSets[metricsetName] = metricSet
	}

	// Setup period
	period, err := time.ParseDuration(m.config.Period)
	if err != nil {
		logp.Info("Error in parsing period of metric %s: %v", m.name, err)
	}

	// If no period set, set default
	if period == 0 {
		logp.Info("Setting default period for metric %s as not set.", m.name)
		period = 1 * time.Second
	}

	// TODO: Improve logging information with list (names of metricSets)
	logp.Info("Start Module %s with metricsets %v and period %v", m.name, m.metricSets, period)

	go m.Run(period, b)
}

func (m *Module) Run(period time.Duration, b *beat.Beat) {
	ticker := time.NewTicker(period)
	defer func() {
		logp.Info("Stopped module %s with metricsets ... TODO", m.name)
		ticker.Stop()
	}()

	var wg sync.WaitGroup
	ch := make(chan struct{})

	wait := func() {
		wg.Wait()
		ch <- struct{}{}
	}

	// TODO: A fetch event should take a maximum until the next ticker and
	// be stopped before the next request is sent. If a fetch is not successful
	// until the next it means it is a failure and a "error" event should be sent to es
	fetch := func(set *MetricSet) {
		defer wg.Done()
		// Move execution part to module?
		m.FetchMetricSets(b, set)
	}

	for {
		// Waits for next ticker
		select {
		case <-m.done:
			return
		case <-ticker.C:
		}

		for _, set := range m.metricSets {
			wg.Add(1)
			go fetch(set)
		}
		go wait()

		// Waits until all fetches are finished
		select {
		case <-m.done:
			return
		case <-ch:
			// finished parallel fetch
		}
	}
}

func (m *Module) FetchMetricSets(b *beat.Beat, metricSet *MetricSet) {

	m.wg.Add(1)

	// Catches metric in case of panic. Keeps other metricsets running
	defer m.wg.Done()

	// Separate defer call as is has to be called directly
	defer logp.Recover(fmt.Sprintf("Metric %s paniced and stopped running.", m.name))

	events, err := metricSet.Fetch()

	if err != nil {
		// TODO: Also list module?
		logp.Err("Fetching events in MetricSet %s returned error: %s", metricSet.Name, err)
		// TODO: Still publish event with error
		return
	}

	// Async publishing of event
	b.Events.PublishEvents(events)

}

// Stop stops module and all its metricSets
func (m *Module) Stop() {
	logp.Info("Stopping module: %v", m.name)
	m.wg.Wait()
}
