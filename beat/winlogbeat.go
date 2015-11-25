package beat

import (
	"expvar"
	"fmt"
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

// Debug logging functions for this package.
var (
	debugf    = logp.MakeDebug("winlogbeat")
	detailf   = logp.MakeDebug("winlogbeat_detail")
	memstatsf = logp.MakeDebug("memstats")
)

// Time the application was started.
var startTime = time.Now().UTC()

func init() {
	expvar.Publish("uptime", expvar.Func(uptime))
}

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
		logp.Err("Error reading configuration file. %v", err)
		return err
	}

	// Validate configuration.
	err = eb.config.Winlogbeat.Validate()
	if err != nil {
		logp.Err("Error validating configuration file. %v", err)
		return err
	}
	debugf("Init winlogbeat. Config: %v", eb.config)

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
				logp.Warn("Unable to launch HTTP service for metrics. %v", err)
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

	// Initialize metrics.
	publishedEvents.Add("total", 0)
	publishedEvents.Add("failures", 0)
	ignoredEvents.Add("total", 0)

	var wg sync.WaitGroup

	// TODO: If no event_logs are specified in the configuration, use the
	// Windows registry to discover the available event logs.
	for _, eventLogConfig := range eb.config.Winlogbeat.EventLogs {
		debugf("Initializing EventLog[%s]", eventLogConfig.Name)

		eventLogAPI := eventlog.NewEventLoggingAPI(eventLogConfig.Name)
		eb.eventLogs = append(eb.eventLogs, eventLogAPI)
		ignoreOlder, _ := config.IgnoreOlderDuration(eventLogConfig.IgnoreOlder)

		// Initialize per event log metrics.
		publishedEvents.Add(eventLogConfig.Name, 0)
		ignoredEvents.Add(eventLogConfig.Name, 0)

		// Start a goroutine for each event log.
		wg.Add(1)
		go eb.processEventLog(&wg, eventLogAPI, ignoreOlder)
	}

	wg.Wait()
	return nil
}

func (eb *Winlogbeat) Cleanup(b *beat.Beat) error {
	logp.Info("Dumping runtime metrics...")
	expvar.Do(func(kv expvar.KeyValue) {
		logf := logp.Info
		if kv.Key == "memstats" {
			logf = memstatsf
		}

		logf("%s=%s", kv.Key, kv.Value.String())
	})
	return nil
}

func (eb *Winlogbeat) Stop() {
	logp.Info("Initiating shutdown, please wait.")
	close(eb.done)
}

func (eb *Winlogbeat) processEventLog(
	wg *sync.WaitGroup,
	api eventlog.EventLoggingAPI,
	ignoreOlder time.Duration,
) {
	defer wg.Done()

	err := api.Open(0)
	if err != nil {
		logp.Warn("EventLog[%s] Open() error. No events will be read from "+
			"this source. %v", api.Name(), err)
		return
	}
	defer func() {
		err := api.Close()
		if err != nil {
			logp.Warn("EventLog[%s] Close() error. %v", api.Name(), err)
			return
		}
	}()

	debugf("EventLog[%s] opened successfully", api.Name())

loop:
	for {
		select {
		case <-eb.done:
			break loop
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

		// Filter events.
		var events []common.MapStr
		for _, lr := range records {
			// TODO: Move filters close to source. Short circuit processing
			// of event if it is going to be filtered.
			// TODO: Add a severity filter.
			// TODO: Check the global IgnoreOlder filter.
			if ignoreOlder != 0 && time.Since(lr.TimeGenerated) > ignoreOlder {
				detailf("EventLog[%s] ignore_older filter dropping event: %s",
					api.Name(), lr.String())
				ignoredEvents.Add("total", 1)
				ignoredEvents.Add(api.Name(), 1)
				continue
			}

			events = append(events, lr.ToMapStr())
		}

		// Publish events.
		numEvents := int64(len(events))
		ok := eb.client.PublishEvents(events, publisher.Sync)
		if ok {
			publishedEvents.Add("total", numEvents)
			publishedEvents.Add(api.Name(), numEvents)
			logp.Info("EventLog[%s] Successfully published %d events", api.Name(), numEvents)
		} else {
			logp.Warn("EventLog[%s] Failed to publish %d events", api.Name(), numEvents)
			publishedEvents.Add("failures", 1)
		}
	}
}

// uptime returns a map of uptime related metrics.
func uptime() interface{} {
	now := time.Now().UTC()
	uptimeDur := now.Sub(startTime)

	return map[string]interface{}{
		"start_time":  startTime,
		"uptime":      uptimeDur.String(),
		"uptime_ms":   fmt.Sprintf("%d", uptimeDur.Nanoseconds()/int64(time.Microsecond)),
		"server_time": now,
	}
}
