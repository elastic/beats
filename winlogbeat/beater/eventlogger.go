package beater

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"

	"github.com/elastic/beats/winlogbeat/checkpoint"
	"github.com/elastic/beats/winlogbeat/eventlog"
)

type eventLogger struct {
	source     eventlog.EventLog
	eventMeta  common.EventMetadata
	processors beat.ProcessorList
}

type eventLoggerConfig struct {
	common.EventMetadata `config:",inline"`      // Fields and tags to add to events.
	Processors           processors.PluginConfig `config:"processors"`
}

func newEventLogger(
	source eventlog.EventLog,
	options *common.Config,
) (*eventLogger, error) {
	config := eventLoggerConfig{}
	if err := options.Unpack(&config); err != nil {
		return nil, err
	}

	processors, err := processors.New(config.Processors)
	if err != nil {
		return nil, err
	}

	return &eventLogger{
		source:     source,
		eventMeta:  config.EventMetadata,
		processors: processors,
	}, nil
}

func (e *eventLogger) connect(pipeline beat.Pipeline) (beat.Client, error) {
	api := e.source.Name()
	return pipeline.ConnectWith(beat.ClientConfig{
		PublishMode:   beat.GuaranteedSend,
		EventMetadata: e.eventMeta,
		Meta:          nil, // TODO: configure modules/ES ingest pipeline?
		Processor:     e.processors,
		ACKCount: func(n int) {
			addPublished(api, n)
			logp.Info("EventLog[%s] successfully published %d events", api, n)
		},
	})
}

func (e *eventLogger) run(
	done <-chan struct{},
	pipeline beat.Pipeline,
	state checkpoint.EventLogState,
) {
	api := e.source

	// Initialize per event log metrics.
	initMetrics(api.Name())

	client, err := e.connect(pipeline)
	if err != nil {
		logp.Warn("EventLog[%s] Pipeline error. Failed to connect to publisher pipeline",
			api.Name())
		return
	}

	// close client on function return or when `done` is triggered (unblock client)
	defer client.Close()
	go func() {
		<-done
		client.Close()
	}()

	err = api.Open(state)
	if err != nil {
		logp.Warn("EventLog[%s] Open() error. No events will be read from "+
			"this source. %v", api.Name(), err)
		return
	}
	defer func() {
		logp.Info("EventLog[%s] Stop processing.", api.Name())

		if err := api.Close(); err != nil {
			logp.Warn("EventLog[%s] Close() error. %v", api.Name(), err)
			return
		}
	}()

	debugf("EventLog[%s] opened successfully", api.Name())

	for {
		select {
		case <-done:
			return
		default:
		}

		// Read from the event.
		records, err := api.Read()
		if err != nil {
			logp.Warn("EventLog[%s] Read() error: %v", api.Name(), err)
			break
		}

		debugf("EventLog[%s] Read() returned %d records", api.Name(), len(records))
		if len(records) == 0 {
			// TODO: Consider implementing notifications using
			// NotifyChangeEventLog instead of polling.
			time.Sleep(time.Second)
			continue
		}

		for _, lr := range records {
			client.Publish(lr.ToEvent())
		}
	}
}
