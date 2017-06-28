package file

import (
	"os"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

const (
	metricsetName = "audit.file"
	logPrefix     = "[" + metricsetName + "]"
)

var (
	debugf = logp.MakeDebug(metricsetName)
)

func init() {
	if err := mb.Registry.AddMetricSet("audit", "file", New, parse.EmptyHostParser); err != nil {
		panic(err)
	}
}

type Config struct {
	Paths       map[string][]string `config:"file.paths"`
	MaxFileSize int64               `config:"max_file_size"`
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
	logp.Experimental("The %v metricset is an experimental feature", metricsetName)

	config := Config{
		MaxFileSize: 1 << 30, // 1 Gibibyte
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	r, err := NewEventReader(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize audit file event reader")
	}

	debugf("%v Initialized the audit file event reader. Running as euid=%v",
		logPrefix, os.Geteuid())

	return &MetricSet{BaseMetricSet: base, config: config, reader: r}, nil
}

func (ms *MetricSet) Run(reporter mb.PushReporter) {
	eventChan, err := ms.reader.Start(reporter.Done())
	if err != nil {
		err = errors.Wrap(err, "failed to start event reader")
		reporter.Error(err)
		logp.Err("%v %v", logPrefix, err)
		return
	}

	for {
		select {
		case <-reporter.Done():
			return
		case event := <-eventChan:
			reporter.Event(buildMapStr(&event))
		}
	}
}
