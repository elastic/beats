package beater

import (
	"expvar"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/filter"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
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

// moduleWrapper contains the Module and the private data associated with
// running the Module and its MetricSets. The moduleWrapper contains a list
// of pointer to its associated MetricSets.
type moduleWrapper struct {
	mb.Module
	filters    *filter.FilterList
	pubClient  publisher.Client
	metricSets []*metricSetWrapper
}

// metricSetWrapper contains the MetricSet and the private data associated with
// running the MetricSet. It contains a pointer to the parent Module.
type metricSetWrapper struct {
	mb.MetricSet
	module *moduleWrapper // Parent Module.
	stats  *expvar.Map    // expvar stats for this MetricSet.
}

// newModuleWrappers creates new Modules and their associated MetricSets based
// on the given configuration. It constructs the supporting filters and
// publisher client and stores it all in a moduleWrapper.
func newModuleWrappers(
	modulesConfig []*common.Config,
	r *mb.Register,
	publisher *publisher.Publisher,
) ([]*moduleWrapper, error) {
	modules, err := mb.NewModules(modulesConfig, r)
	if err != nil {
		return nil, err
	}

	// Wrap the Modules and MetricSet's.
	var wrappers []*moduleWrapper
	var errs multierror.Errors
	for k, v := range modules {
		debugf("Initializing Module type '%s': %T=%+v", k.Name(), k, k)
		f, err := filter.New(k.Config().Filters)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "module %s", k.Name()))
			continue
		}

		mw := &moduleWrapper{
			Module:    k,
			filters:   f,
			pubClient: publisher.Connect(),
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

// metricSetWrapper methods

// startFetching performs an immediate fetch for the specified host then it
// begins continuous timer scheduled loop to fetch data. To stop the loop the
// done channel should be closed. On exit the method will decrement the
// WaitGroup counter.
//
// startFetching manages fetching for a single host so it should be called once
// per host.
func (msw *metricSetWrapper) startFetching(
	done <-chan struct{},
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	debugf("Starting %s", msw)
	defer debugf("Stopped %s", msw)

	// Fetch immediately.
	err := msw.fetch()
	if err != nil {
		logp.Err("fetch error: %v", err)
	}

	// Start timer for future fetches.
	t := time.NewTicker(msw.Module().Config().Period)
	defer t.Stop()
	for {
		select {
		case <-done:
			return
		case <-t.C:
			err := msw.fetch()
			if err != nil {
				logp.Err("%v", err)
			}
		}
	}
}

// fetch invokes the appropriate Fetch method for the MetricSet and publishes
// the result using the publisher client. This method will recover from panics
// and log a stack track if one occurs.
func (msw *metricSetWrapper) fetch() error {
	defer logp.Recover(fmt.Sprintf("recovered from panic while fetching "+
		"'%s/%s' for host '%s'", msw.module.Name(), msw.Name(), msw.Host()))

	switch fetcher := msw.MetricSet.(type) {
	case mb.EventFetcher:
		return msw.singleEventFetch(fetcher)
	case mb.EventsFetcher:
		return msw.multiEventFetch(fetcher)
	default:
		return fmt.Errorf("MetricSet '%s/%s' does not implement a Fetcher "+
			"interface", msw.Module().Name(), msw.Name())
	}
}

func (msw *metricSetWrapper) singleEventFetch(fetcher mb.EventFetcher) error {
	start := time.Now()
	event, err := fetcher.Fetch()
	elapsed := time.Since(start)
	if err == nil {
		msw.stats.Add(successesKey, 1)
	} else {
		msw.stats.Add(failuresKey, 1)
	}

	event, err = createEvent(msw, event, err, start, elapsed)
	if err != nil {
		logp.Warn("createEvent error: %v", err)
	}

	if event != nil {
		msw.module.pubClient.PublishEvent(event)
		msw.stats.Add(eventsKey, 1)
	}

	return nil
}

func (msw *metricSetWrapper) multiEventFetch(fetcher mb.EventsFetcher) error {
	start := time.Now()
	events, err := fetcher.Fetch()
	elapsed := time.Since(start)
	if err == nil {
		msw.stats.Add(successesKey, 1)

		for _, event := range events {
			event, err = createEvent(msw, event, nil, start, elapsed)
			if err != nil {
				logp.Warn("createEvent error: %v", err)
			}

			if event != nil {
				msw.module.pubClient.PublishEvent(event)
				msw.stats.Add(eventsKey, 1)
			}
		}
	} else {
		msw.stats.Add(failuresKey, 1)

		event, err := createEvent(msw, nil, err, start, elapsed)
		if err != nil {
			logp.Warn("createEvent error: %v", err)
		}

		if event != nil {
			msw.module.pubClient.PublishEvent(event)
			msw.stats.Add(eventsKey, 1)
		}
	}

	return nil
}

func (msw *metricSetWrapper) String() string {
	return fmt.Sprintf("metricSetWrapper[module=%s, name=%s, host=%s]",
		msw.module.Name(), msw.Name(), msw.Host())
}

// other utility functions

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
