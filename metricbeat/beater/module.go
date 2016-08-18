package beater

import (
	"expvar"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
)

// Expvar metric names.
const (
	successesKey = "success"
	failuresKey  = "failures"
	eventsKey    = "events"
)

var (
	debugf      = logp.MakeDebug("metricbeat")
	fetchesLock = sync.Mutex{}
	fetches     = expvar.NewMap("fetches")
)

// ModuleWrapper contains the Module and the private data associated with
// running the Module and its MetricSets.
//
// Use NewModuleWrapper or NewModuleWrappers to construct new ModuleWrappers.
type ModuleWrapper struct {
	mb.Module
	filters    *processors.Processors
	metricSets []*metricSetWrapper // List of pointers to its associated MetricSets.
}

// metricSetWrapper contains the MetricSet and the private data associated with
// running the MetricSet. It contains a pointer to the parent Module.
type metricSetWrapper struct {
	mb.MetricSet
	module *ModuleWrapper // Parent Module.
	stats  *expvar.Map    // expvar stats for this MetricSet.
}

// NewModuleWrapper create a new Module and its associated MetricSets based
// on the given configuration. It constructs the supporting filters and stores
// them in the ModuleWrapper.
func NewModuleWrapper(moduleConfig *common.Config, r *mb.Register) (*ModuleWrapper, error) {
	mws, err := NewModuleWrappers([]*common.Config{moduleConfig}, r)
	if err != nil {
		return nil, err
	}

	if len(mws) == 0 {
		return nil, fmt.Errorf("module not created")
	}

	return mws[0], nil
}

// NewModuleWrappers creates new Modules and their associated MetricSets based
// on the given configuration. It constructs the supporting filters and stores
// them all in a ModuleWrapper.
func NewModuleWrappers(modulesConfig []*common.Config, r *mb.Register) ([]*ModuleWrapper, error) {
	modules, err := mb.NewModules(modulesConfig, r)
	if err != nil {
		return nil, err
	}

	// Wrap the Modules and MetricSet's.
	var wrappers []*ModuleWrapper
	var errs multierror.Errors
	for k, v := range modules {
		debugf("Initializing Module type '%s': %T=%+v", k.Name(), k, k)
		f, err := processors.New(k.Config().Filters)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "module %s", k.Name()))
			continue
		}

		mw := &ModuleWrapper{
			Module:  k,
			filters: f,
		}
		wrappers = append(wrappers, mw)

		msws := make([]*metricSetWrapper, 0, len(v))
		for _, ms := range v {
			debugf("Initializing MetricSet type '%s/%s' for host '%s': %T=%+v",
				ms.Module().Name(), ms.Name(), ms.Host(), ms, ms)

			expMap, err := getMetricSetExpvarMap(mw.Name(), ms.Name())
			if err != nil {
				return nil, err
			}

			msw := &metricSetWrapper{
				MetricSet: ms,
				module:    mw,
				stats:     expMap,
			}
			msws = append(msws, msw)
		}
		mw.metricSets = msws
	}

	return wrappers, errs.Err()
}

// ModuleWrapper methods

// Start starts the Module's MetricSet workers which are responsible for
// fetching metrics. The workers will continue to periodically fetch until the
// done channel is closed. When the done channel is closed all MetricSet workers
// will stop and the returned output channel will be closed.
//
// The returned channel is buffered with a length one one. It must drained to
// prevent blocking the operation of the MetricSets.
//
// Start should be called only once in the life of a ModuleWrapper.
func (mw *ModuleWrapper) Start(done <-chan struct{}) <-chan common.MapStr {
	debugf("Starting %s", mw)
	defer debugf("Stopped %s", mw)

	out := make(chan common.MapStr, 1)

	// Start one worker per MetricSet + host combination.
	var wg sync.WaitGroup
	wg.Add(len(mw.metricSets))
	for _, msw := range mw.metricSets {
		go func(msw *metricSetWrapper) {
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

// String returns a string representation of ModuleWrapper.
func (mw *ModuleWrapper) String() string {
	return fmt.Sprintf("ModuleWrapper[name=%s, len(metricSetWrappers)=%d]",
		mw.Name(), len(mw.metricSets))
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
			msw.stats.Add(eventsKey, 1)
			writeEvent(done, out, event)
		}
	case mb.EventsFetcher:
		events, err := msw.multiEventFetch(fetcher)
		if err != nil {
			return err
		}
		for _, event := range events {
			msw.stats.Add(eventsKey, 1)
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
		msw.stats.Add(successesKey, 1)
	} else {
		msw.stats.Add(failuresKey, 1)
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
		msw.stats.Add(successesKey, 1)

		for _, event := range events {
			if event, err = createEvent(msw, event, nil, start, elapsed); err != nil {
				return nil, errors.Wrap(err, "createEvent failed")
			}
			if event != nil {
				rtnEvents = append(rtnEvents, event)
			}
		}
	} else {
		msw.stats.Add(failuresKey, 1)

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

func getMetricSetExpvarMap(module, name string) (*expvar.Map, error) {
	key := fmt.Sprintf("%s-%s", module, name)
	fetchesLock.Lock()
	defer fetchesLock.Unlock()

	expVar := fetches.Get(key)
	switch m := expVar.(type) {
	case nil:
		expMap := new(expvar.Map).Init()
		fetches.Set(key, expMap)
		expMap.Add(successesKey, 0)
		expMap.Add(failuresKey, 0)
		expMap.Add(eventsKey, 0)
		return expMap, nil
	case *expvar.Map:
		return m, nil
	default:
		return nil, fmt.Errorf("unexpected expvar.Var type (%T) found for key '%s'", m, key)
	}
}
