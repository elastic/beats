package file

import (
	"os"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

const (
	metricsetName = "audit.file"
)

var (
	debugf = logp.MakeDebug(metricsetName)
)

func init() {
	if err := mb.Registry.AddMetricSet("audit", "file", New, parse.EmptyHostParser); err != nil {
		panic(err)
	}
}

type EventReader interface {
	Start(done <-chan struct{}) (<-chan Event, error)
}

type MetricSet struct {
	mb.BaseMetricSet
	config Config
	reader EventReader
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The %v metricset is an experimental feature", metricsetName)

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	r, err := NewEventReader(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize audit file event reader")
	}

	debugf("Initialized the audit file event reader. Running as euid=%v", os.Geteuid())

	return &MetricSet{BaseMetricSet: base, config: config, reader: r}, nil
}

func (ms *MetricSet) Run(reporter mb.PushReporter) {
	eventChan, err := ms.reader.Start(reporter.Done())
	if err != nil {
		err = errors.Wrap(err, "failed to start event reader")
		reporter.Error(err)
		logp.Err("%v", err)
		return
	}

	for {
		select {
		case <-reporter.Done():
			return
		case event := <-eventChan:
			reporter.Event(buildMapStr(&event))

			if len(event.errors) > 0 {
				debugf("Errors on %v event for %v: %v",
					event.Action, event.Path, event.errors)
			}
		}
	}
}
