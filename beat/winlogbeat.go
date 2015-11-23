package beat

import (
	"expvar"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/elastic/libbeat/beat"
	"github.com/elastic/libbeat/cfgfile"
	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/publisher"
	"github.com/elastic/winlogbeat/config"
	"github.com/elastic/winlogbeat/eventlog"
)

// Metrics that can retrieved through the expvar web interface. Metrics must be
// enable through configuration in order for the web service to be started.
var (
	publishedEvents = expvar.NewMap("publishedEvents")
	ignoredEvents   = expvar.NewMap("ignoredEvents")
)

type Winlogbeat struct {
	beat      *beat.Beat                 // Common beat information.
	config    *config.ConfigSettings     // Configuration settings.
	eventLogs []eventlog.EventLoggingAPI // Interface to the event logs.
	done      chan struct{}              // Channel to initiate shutdown of main event loop.
	client    publisher.Client           // Interface to publish event.
}

func (eb *Winlogbeat) Config(b *beat.Beat) error {
	// Read configuration.
	err := cfgfile.Read(&eb.config, "")
	if err != nil {
		logp.Err("Error reading configuration file: %v", err)
		return err
	}

	// Validate configuration.
	err = eb.config.Winlogbeat.Validate()
	if err != nil {
		logp.Err("Error validating configuration file: %v", err)
		return err
	}
	logp.Debug("winlogbeat", "Init winlogbeat. Config: %v", eb.config)

	return nil
}

func (eb *Winlogbeat) Setup(b *beat.Beat) error {
	eb.beat = b
	eb.client = b.Events
	eb.done = make(chan struct{})

	if eb.config.Winlogbeat.Metrics.BindAddress != "" {
		bindAddress := eb.config.Winlogbeat.Metrics.BindAddress
		sock, err := net.Listen("tcp", bindAddress)
		if err != nil {
			return err
		}
		go func() {
			logp.Info("Metrics hosted at http://%s/debug/vars", bindAddress)
			err := http.Serve(sock, nil)
			if err != nil {
				logp.Warn("Unable to launch HTTP service for metrics. err=%v", err)
				return
			}
		}()
	}

	return nil
}

func (eb *Winlogbeat) Run(b *beat.Beat) error {
	// TODO: Persist last published RecordNumber for each event log so that
	// when restarted, winlogbeat resumes from the last read event. This should
	// provide at-least-once publish semantics.

	publishedEvents.Add("total", 0)
	ignoredEvents.Add("total", 0)

	var wg sync.WaitGroup

	// TODO: If no event_logs are specified in the configuration, use the
	// Windows registry to discover the available event logs.
	for _, eventLogConfig := range eb.config.Winlogbeat.EventLogs {
		logp.Debug("winlogbeat", "Creating event log for %s.",
			eventLogConfig.Name)
		eventLogAPI := eventlog.NewEventLoggingAPI(eventLogConfig.Name)
		ignoreOlder, _ := config.IgnoreOlderDuration(eventLogConfig.IgnoreOlder)
		eb.eventLogs = append(eb.eventLogs, eventLogAPI)
		publishedEvents.Add(eventLogConfig.Name, 0)
		publishedEvents.Add("failures", 0)
		ignoredEvents.Add(eventLogConfig.Name, 0)

		go func(api eventlog.EventLoggingAPI, ignoreOlder time.Duration) {
			err := api.Open(0)
			if err != nil {
				logp.Warn("EventLog[%s] Open() error: %v", api.Name(), err)
				wg.Done()
				return
			}
			defer func() {
				err := api.Close()
				if err != nil {
					logp.Warn("EventLog[%s] Close() error: %v", api.Name(), err)
					return
				}
			}()

			logp.Debug("winlogbeat", "EventLog[%s] opened successfully",
				api.Name())

		loop:
			for {
				select {
				case <-eb.done:
					break loop
				default:
				}

				records, err := api.Read()
				if err != nil {
					logp.Warn("EventLog[%s] Read() error: %v", api.Name(), err)
					break
				}

				logp.Debug("winlogbeat", "EventLog[%s] Read() returned %d "+
					"records.", api.Name(), len(records))
				if len(records) == 0 {
					time.Sleep(time.Second)
					continue
				}

				var events []common.MapStr
				for _, lr := range records {
					// TODO: Move filters close to source. Short circuit processing
					// of event if it is going to be filtered.
					// TODO: Add a severity filter.
					// TODO: Check the global IgnoreOlder filter.
					if ignoreOlder != 0 && time.Since(lr.TimeGenerated) > ignoreOlder {
						logp.Debug("winlogbeat", "ignoreOlder filter dropping "+
							"event: %s", lr.String())
						ignoredEvents.Add("total", 1)
						ignoredEvents.Add(api.Name(), 1)
						continue
					}

					events = append(events, lr.ToMapStr())
				}

				numEvents := int64(len(events))
				ok := eb.client.PublishEvents(events, publisher.Sync)
				if ok {
					publishedEvents.Add("total", numEvents)
					publishedEvents.Add(api.Name(), numEvents)
					logp.Debug("winlogbeat", "EvengLog[%s] Successfully "+
						"published %d events.", api.Name(), numEvents)
				} else {
					logp.Warn("winlogbeat", "EventLog[%s] Failed to publish %d "+
						"events.", api.Name(), numEvents)
					publishedEvents.Add("failures", 1)
				}
			}

			wg.Done()
		}(eventLogAPI, ignoreOlder)

		wg.Add(1)
	}

	wg.Wait()
	return nil
}

func (eb *Winlogbeat) Cleanup(b *beat.Beat) error {
	logp.Debug("winlogbeat", "Dumping runtime metrics...")
	expvar.Do(func(kv expvar.KeyValue) {
		logp.Debug("winlogbeat", "%s=%s", kv.Key, kv.Value.String())
	})
	return nil
}

func (eb *Winlogbeat) Stop() {
	logp.Info("Initiating shutdown, please wait.")
	close(eb.done)
}
