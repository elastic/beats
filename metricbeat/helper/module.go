package helper

import (
	"sort"
	"sync"
	"time"

	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/filter"
	"github.com/elastic/beats/libbeat/logp"
)

// Module specifics. This must be defined by each module
type Module struct {
	name string

	// Moduler implementation
	moduler Moduler

	// Module config
	Config ModuleConfig

	Timeout time.Duration

	// Raw config object to be unpacked by moduler
	cfg *common.Config

	// List of all metricsets in this module. Use to keep track of metricsets
	metricSets map[string]*MetricSet

	Publish chan common.MapStr

	wg      sync.WaitGroup // MetricSet waitgroup
	done    chan struct{}
	filters *filter.FilterList
}

// NewModule creates a new module
func NewModule(cfg *common.Config, moduler func() Moduler) (*Module, error) {

	// Module config defaults
	config := ModuleConfig{
		Period:  "1s",
		Enabled: true,
	}

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	filters, err := filter.New(config.Filters)
	if err != nil {
		return nil, fmt.Errorf("error initializing filters: %v", err)
	}
	logp.Debug("module", "Filters: %+v", filters)

	return &Module{
		name:       config.Module,
		Config:     config,
		cfg:        cfg,
		moduler:    moduler(),
		metricSets: map[string]*MetricSet{},
		Publish:    make(chan common.MapStr), // TODO: What should be size of channel? @ruflin,20160316
		wg:         sync.WaitGroup{},
		done:       make(chan struct{}),
		filters:    filters,
	}, nil
}

// Starts the given module
func (m *Module) Start(b *beat.Beat) error {

	defer logp.Recover(fmt.Sprintf("Module %s paniced and stopped running.", m.name))

	if !m.Config.Enabled {
		logp.Debug("helper", "Not starting module %s with metricsets %s as not enabled.", m.name, m.getMetricSetsList())
		return nil
	}

	logp.Info("Setup moduler: %s", m.name)
	err := m.moduler.Setup(m)
	if err != nil {
		return fmt.Errorf("Error setting up module: %s. Not starting metricsets for this module.", err)
	}

	err = m.loadMetricsets()
	if err != nil {
		return fmt.Errorf("Error loading metricsets: %s", err)
	}

	// Setup period
	period, err := time.ParseDuration(m.Config.Period)
	if err != nil {
		return fmt.Errorf("Error in parsing period of module %s: %v", m.name, err)
	}

	// If no period set, set default
	if period == 0 {
		logp.Info("Setting default period for module %s as not set.", m.name)
		period = 1 * time.Second
	}

	var timeout time.Duration

	if m.Config.Timeout != "" {
		// Setup timeout
		timeout, err := time.ParseDuration(m.Config.Timeout)
		if err != nil {
			return fmt.Errorf("Error in parsing timeout of module %s: %v", m.name, err)
		}

		// If no timeout set, set to period as default
		if timeout == 0 {
			logp.Info("Setting default timeout for module %s as not set.", m.name)
			timeout = period
		}
	} else {
		timeout = period
	}
	m.Timeout = timeout

	logp.Info("Start Module %s with metricsets [%s] and period %v", m.name, m.getMetricSetsList(), period)

	m.setupMetricSets()

	go m.Run(period, b)

	return nil
}

func (m *Module) setupMetricSets() {
	for _, set := range m.metricSets {
		err := set.Setup()
		if err != nil {
			logp.Err("Error setting up MetricSet %s: %s", set.Name, err)
		}
	}
}

func (m *Module) Run(period time.Duration, b *beat.Beat) {
	ticker := time.NewTicker(period)

	defer func() {
		logp.Info("Stopped module %s with metricsets %s", m.name, m.getMetricSetsList())
		ticker.Stop()
	}()

	fetch := func(set *MetricSet) {
		m.FetchMetricSets(set)
	}

	// Start publisher
	go m.publishing(b)

	// Start fetching metrics
	// The frequency is always based on the ticker interval. No delays are taken into account.
	// Each Metricset must ensure to exit after timeout.
	for {
		for _, set := range m.metricSets {
			go fetch(set)
		}

		// Waits for next ticker
		select {
		case <-m.done:
			return
		case <-ticker.C:
		}
	}
}

func (m *Module) FetchMetricSets(metricSet *MetricSet) {

	// Separate defer call as is has to be called directly
	defer logp.Recover(fmt.Sprintf("Metric %s paniced and stopped running.", m.name))

	err := metricSet.Fetch()
	if err != nil {
		logp.Err("Fetching events for MetricSet %s in Module %s returned error: %s", metricSet.Name, m.name, err)
	}
}

// Stop stops module and all its metricSets
func (m *Module) Stop() {
	logp.Info("Stopping module: %v", m.name)
	m.wg.Wait()
}

// loadMetricsets creates and setups the metricseter for the module
func (m *Module) loadMetricsets() error {
	// Setup - Create metricSets for the module
	for _, metricsetName := range m.Config.MetricSets {

		metricSet, err := Registry.GetMetricSet(m, metricsetName)
		if err != nil {
			return err
		}
		m.metricSets[metricsetName] = metricSet
	}
	return nil
}

// getMetricSetsList is a helper function that returns a list of all module metricsets as string
// This is mostly used for logging
func (m *Module) getMetricSetsList() string {

	// Sort list first alphabetically
	keys := make([]string, 0, len(m.metricSets))
	for key := range m.metricSets {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Create output string
	list := ""
	first := true

	for _, value := range keys {
		if !first {
			list = list + ", "
		}
		first = false
		list = list + value
	}

	return list
}

// publishing runs the receiving channel to receive events from the metricset
// and forward them to the publisher
func (m *Module) publishing(b *beat.Beat) {
	for {

		select {
		case <-m.done:
			return
		case event := <-m.Publish:
			// TODO transform to publish events - @ruflin,20160314
			// Will this merge multiple events together to use bulk sending?
			b.Events.PublishEvent(event)
		}
	}
}

// ProcessConfig allows to process additional configuration params which are not
// part of the module default configuratoin. This allows each metricset
// to have its specific config params
func (m *Module) ProcessConfig(config interface{}) error {

	if err := m.cfg.Unpack(config); err != nil {
		return err
	}
	return nil
}
