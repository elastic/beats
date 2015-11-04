package beat

import (
	"expvar"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/elastic/eventbeat/config"
	"github.com/elastic/eventbeat/eventlog"
	"github.com/elastic/libbeat/beat"
	"github.com/elastic/libbeat/cfgfile"
	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/publisher"
)

// Metrics that can retrieved through the expvar web interface. Metrics must be
// enable through configuration in order for the web service to be started.
var (
	publishedEvents = expvar.NewMap("publishedEvents")
	ignoredEvents   = expvar.NewMap("ignoredEvents")
)

type Eventbeat struct {
	beat      *beat.Beat                 // Common beat information.
	config    *config.ConfigSettings     // Configuration settings.
	eventLogs []eventlog.EventLoggingAPI // Interface to the event logs.
	stop      AtomicBool                 // Boolean flag to initiate shutdown of main event loop.
	client    publisher.Client           // Interface to publish event.
}

func (eb *Eventbeat) Config(b *beat.Beat) error {
	// Read configuration.
	err := cfgfile.Read(&eb.config, "")
	if err != nil {
		logp.Err("Error reading configuration file: %v", err)
		return err
	}

	// Validate configuration.
	err = eb.config.Eventbeat.Validate()
	if err != nil {
		logp.Err("Error validating configuration file: %v", err)
		return err
	}
	logp.Debug("eventbeat", "Init eventbeat. Config: %v", eb.config)

	return nil
}

func (eb *Eventbeat) Setup(b *beat.Beat) error {
	eb.beat = b
	eb.client = b.Events

	// If metrics are enabled, host expvars at http://<bindaddress>/debug/vars.
	if eb.config.Eventbeat.Metrics.BindAddress != "" {
		bindAddress := eb.config.Eventbeat.Metrics.BindAddress
		sock, err := net.Listen("tcp", bindAddress)
		if err != nil {
			return err
		}
		go func() {
			logp.Info("HTTP now available at %s", bindAddress)
			http.Serve(sock, nil)
		}()
	}

	return nil
}

func (eb *Eventbeat) Run(b *beat.Beat) error {
	// TODO: Persist last published RecordNumber for each event log so that
	// when restarted, eventbeat resumes from the last read event. This should
	// provide at-least-once publish semantics.

	publishedEvents.Add("total", 0)
	ignoredEvents.Add("total", 0)

	var wg sync.WaitGroup

	// TODO: If no event_logs are specified in the configuration, use the
	// Windows registry to discover the available event logs.
	for _, eventLogConfig := range eb.config.Eventbeat.EventLogs {
		logp.Debug("eventbeat", "Creating event log for %s.",
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
			defer api.Close()

			logp.Debug("eventlog", "EventLog[%s] opened successfully",
				api.Name())

			for !eb.stop.Get() {
				records, err := api.Read()
				if err != nil {
					logp.Warn("EventLog[%s] Read() error: %v", api.Name(), err)
					break
				}

				logp.Debug("eventbeat", "EventLog[%s] Read() returned %d "+
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
						logp.Debug("eventbeat", "ignoreOlder filter dropping "+
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
					logp.Debug("eventbeat", "EvengLog[%s] Successfully "+
						"published %d events.", api.Name(), numEvents)
				} else {
					logp.Warn("eventbeat", "EventLog[%s] Failed to publish %d "+
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

func (eb *Eventbeat) Cleanup(b *beat.Beat) error {
	logp.Debug("eventbeat", "Dumping runtime metrics...")
	expvar.Do(func(kv expvar.KeyValue) {
		logp.Debug("eventbeat", "%s=%s", kv.Key, kv.Value.String())
	})
	return nil
}

func (eb *Eventbeat) Stop() {
	logp.Info("Initiating shutdown, please wait.")
	// TODO: Remove atomic bool and use a channel to signal shutdown. Caution:
	// Stop() can be invoked more than once on Windows when you Ctrl+C (one
	// callback for svc shutdown and one for the Ctrl+C) which might cause a
	// double golang channel close bug.
	eb.stop.Set(true)
}
