package module

import (
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/joeshaw/multierror"
	"github.com/mitchellh/hashstructure"
	"github.com/pkg/errors"
)

// Expvar metric names.
const (
	successesKey = "success"
	failuresKey  = "failures"
	eventsKey    = "events"
)

var (
	debugf = logp.MakeDebug("module")

	fetchesLock = sync.Mutex{}
	fetches     = map[string]*stats{}
)

// Wrapper contains the Module and the private data associated with
// running the Module and its MetricSets.
//
// Use NewWrapper or NewWrappers to construct new Wrappers.
type Wrapper struct {
	mb.Module
	filters    *processors.Processors
	metricSets []*metricSetWrapper // List of pointers to its associated MetricSets.
	configHash uint64
}

// metricSetWrapper contains the MetricSet and the private data associated with
// running the MetricSet. It contains a pointer to the parent Module.
type metricSetWrapper struct {
	mb.MetricSet
	module *Wrapper // Parent Module.
	stats  *stats   // stats for this MetricSet.
}

// stats bundles common metricset stats
type stats struct {
	key      string // full stats key
	ref      uint32 // number of modules/metricsets reusing stats instance
	success  *monitoring.Int
	failures *monitoring.Int
	events   *monitoring.Int
}

// NewWrapper create a new Module and its associated MetricSets based
// on the given configuration. It constructs the supporting filters and stores
// them in the Wrapper.
func NewWrapper(moduleConfig *common.Config, r *mb.Register) (*Wrapper, error) {
	mws, err := NewWrappers([]*common.Config{moduleConfig}, r)
	if err != nil {
		return nil, err
	}

	if len(mws) == 0 {
		return nil, fmt.Errorf("module not created")
	}

	return mws[0], nil
}

// NewWrappers creates new Modules and their associated MetricSets based
// on the given configuration. It constructs the supporting filters and stores
// them all in a Wrapper.
func NewWrappers(modulesConfig []*common.Config, r *mb.Register) ([]*Wrapper, error) {
	modules, err := mb.NewModules(modulesConfig, r)
	if err != nil {
		return nil, err
	}

	// Wrap the Modules and MetricSet's.
	var wrappers []*Wrapper
	var errs multierror.Errors
	for k, v := range modules {
		debugf("Initializing Module type '%s': %T=%+v", k.Name(), k, k)
		f, err := processors.New(k.Config().Filters)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "module %s", k.Name()))
			continue
		}

		mw := &Wrapper{
			Module:  k,
			filters: f,
		}
		wrappers = append(wrappers, mw)

		msws := make([]*metricSetWrapper, 0, len(v))
		for _, ms := range v {
			debugf("Initializing MetricSet type '%s/%s' for host '%s': %T=%+v",
				ms.Module().Name(), ms.Name(), ms.Host(), ms, ms)

			msw := &metricSetWrapper{
				MetricSet: ms,
				module:    mw,
				stats:     getMetricSetStats(mw.Name(), ms.Name()),
			}
			msws = append(msws, msw)
		}
		mw.metricSets = msws
	}

	return wrappers, errs.Err()
}

// Wrapper methods

// Start starts the Module's MetricSet workers which are responsible for
// fetching metrics. The workers will continue to periodically fetch until the
// done channel is closed. When the done channel is closed all MetricSet workers
// will stop and the returned output channel will be closed.
//
// The returned channel is buffered with a length one one. It must drained to
// prevent blocking the operation of the MetricSets.
//
// Start should be called only once in the life of a Wrapper.
func (mw *Wrapper) Start(done <-chan struct{}) <-chan common.MapStr {
	debugf("Starting %s", mw)
	defer debugf("Stopped %s", mw)

	out := make(chan common.MapStr, 1)

	// Start one worker per MetricSet + host combination.
	var wg sync.WaitGroup
	wg.Add(len(mw.metricSets))
	for _, msw := range mw.metricSets {
		go func(msw *metricSetWrapper) {
			defer releaseStats(msw.stats)
			defer wg.Done()
			msw.startFetching(done, out)
		}(msw)
	}

	// Close the output channel when all writers to the channel have stopped.
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// String returns a string representation of Wrapper.
func (mw *Wrapper) String() string {
	return fmt.Sprintf("Wrapper[name=%s, len(metricSetWrappers)=%d]",
		mw.Name(), len(mw.metricSets))
}

// Hash returns the hash value of the module wrapper
// This allows to check if two modules are the same / have the same config
func (mw *Wrapper) Hash() uint64 {
	// Check if hash was calculated previously
	if mw.configHash > 0 {
		return mw.configHash
	}
	var err error

	// Config is unpacked into map[string]interface{} to also take metricset configs into account for the hash
	var c map[string]interface{}
	mw.UnpackConfig(&c)
	mw.configHash, err = hashstructure.Hash(c, nil)
	if err != nil {
		logp.Err("Error creating config hash for module %s: %s", mw.String(), err)
	}
	return mw.configHash
}

// metricSetWrapper methods

// startFetching performs an immediate fetch for the MetricSet then it
// begins a continuous timer scheduled loop to fetch data. To stop the loop the
// done channel should be closed.
func (msw *metricSetWrapper) startFetching(
	done <-chan struct{},
	out chan<- common.MapStr,
) {
	debugf("Starting %s", msw)
	defer debugf("Stopped %s", msw)

	// Fetch immediately.
	err := msw.fetch(done, out)
	if err != nil {
		logp.Err("%v", err)
	}

	// Start timer for future fetches.
	t := time.NewTicker(msw.Module().Config().Period)
	defer t.Stop()
	for {
		select {
		case <-done:
			return
		case <-t.C:
			err := msw.fetch(done, out)
			if err != nil {
				logp.Err("%v", err)
			}
		}
	}
}

// fetch invokes the appropriate Fetch method for the MetricSet and publishes
// the result using the publisher client. This method will recover from panics
// and log a stack track if one occurs.
func (msw *metricSetWrapper) fetch(done <-chan struct{}, out chan<- common.MapStr) error {
	defer logp.Recover(fmt.Sprintf("recovered from panic while fetching "+
		"'%s/%s' for host '%s'", msw.module.Name(), msw.Name(), msw.Host()))

	switch fetcher := msw.MetricSet.(type) {
	case mb.EventFetcher:
		event, err := msw.singleEventFetch(fetcher)
		if err != nil {
			return err
		}
		if event != nil {
			msw.stats.events.Add(1)
			writeEvent(done, out, event)
		}
	case mb.EventsFetcher:
		events, err := msw.multiEventFetch(fetcher)
		if err != nil {
			return err
		}
		for _, event := range events {
			msw.stats.events.Add(1)
			if !writeEvent(done, out, event) {
				break
			}
		}
	default:
		return fmt.Errorf("MetricSet '%s/%s' does not implement a Fetcher "+
			"interface", msw.Module().Name(), msw.Name())
	}

	return nil
}

func (msw *metricSetWrapper) singleEventFetch(fetcher mb.EventFetcher) (common.MapStr, error) {
	start := time.Now()
	event, err := fetcher.Fetch()
	elapsed := time.Since(start)

	if err == nil {
		msw.stats.success.Add(1)
	} else {
		msw.stats.failures.Add(1)
	}

	if event, err = createEvent(msw, event, err, start, elapsed); err != nil {
		return nil, errors.Wrap(err, "createEvent failed")
	}

	return event, nil
}

func (msw *metricSetWrapper) multiEventFetch(fetcher mb.EventsFetcher) ([]common.MapStr, error) {
	start := time.Now()
	events, err := fetcher.Fetch()
	elapsed := time.Since(start)

	var rtnEvents []common.MapStr
	if err == nil {
		msw.stats.success.Add(1)

		for _, event := range events {
			if event, err = createEvent(msw, event, nil, start, elapsed); err != nil {
				return nil, errors.Wrap(err, "createEvent failed")
			}
			if event != nil {
				rtnEvents = append(rtnEvents, event)
			}
		}
	} else {
		msw.stats.failures.Add(1)

		event, err := createEvent(msw, nil, err, start, elapsed)
		if err != nil {
			return nil, errors.Wrap(err, "createEvent failed")
		}
		if event != nil {
			rtnEvents = append(rtnEvents, event)
		}
	}

	return rtnEvents, nil
}

// String returns a string representation of metricSetWrapper.
func (msw *metricSetWrapper) String() string {
	return fmt.Sprintf("metricSetWrapper[module=%s, name=%s, host=%s]",
		msw.module.Name(), msw.Name(), msw.Host())
}

// other utility functions

func writeEvent(done <-chan struct{}, out chan<- common.MapStr, event common.MapStr) bool {
	select {
	case <-done:
		return false
	case out <- event:
		return true
	}
}

func getMetricSetStats(module, name string) *stats {
	key := fmt.Sprintf("metricbeat.%s.%s", module, name)

	fetchesLock.Lock()
	defer fetchesLock.Unlock()

	if s := fetches[key]; s != nil {
		s.ref++
		return s
	}

	reg := monitoring.Default.NewRegistry(key)
	s := &stats{
		key:      key,
		ref:      1,
		success:  monitoring.NewInt(reg, successesKey),
		failures: monitoring.NewInt(reg, failuresKey),
		events:   monitoring.NewInt(reg, eventsKey),
	}

	fetches[key] = s
	return s
}

func releaseStats(s *stats) {
	fetchesLock.Lock()
	defer fetchesLock.Unlock()

	s.ref--
	if s.ref > 0 {
		return
	}

	delete(fetches, s.key)
	monitoring.Default.Remove(s.key)
}
