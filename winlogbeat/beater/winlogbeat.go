/*
Package beater provides the implementation of the libbeat Beater interface for
Winlogbeat. The main event loop is implemented in this package.
*/
package beater

import (
	"expvar"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/winlogbeat/checkpoint"
	"github.com/elastic/beats/winlogbeat/config"
	"github.com/elastic/beats/winlogbeat/eventlog"
)

// Metrics that can retrieved through the expvar web interface. Metrics must be
// enable through configuration in order for the web service to be started.
var (
	publishedEvents = expvar.NewMap("published_events")
	ignoredEvents   = expvar.NewMap("ignored_events")
)

func init() {
	expvar.Publish("uptime", expvar.Func(uptime))
}

// Debug logging functions for this package.
var (
	debugf    = logp.MakeDebug("winlogbeat")
	memstatsf = logp.MakeDebug("memstats")
)

// Time the application was started.
var startTime = time.Now().UTC()

// Winlogbeat is used to conform to the beat interface
type Winlogbeat struct {
	beat       *beat.Beat             // Common beat information.
	config     *config.Settings       // Configuration settings.
	eventLogs  []eventlog.EventLog    // List of all event logs being monitored.
	done       chan struct{}          // Channel to initiate shutdown of main event loop.
	client     publisher.Client       // Interface to publish event.
	checkpoint *checkpoint.Checkpoint // Persists event log state to disk.
}

// New returns a new Winlogbeat.
func New(b *beat.Beat, _ *common.Config) (beat.Beater, error) {
	// Read configuration.
	// XXX: winlogbeat validates top-level config -> ignore beater config and
	//      parse complete top-level config
	config := config.DefaultSettings
	rawConfig := b.RawConfig
	err := rawConfig.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("Error reading configuration file. %v", err)
	}

	// reslove registry file path
	config.Winlogbeat.RegistryFile = paths.Resolve(
		paths.Data, config.Winlogbeat.RegistryFile)
	logp.Info("State will be read from and persisted to %s",
		config.Winlogbeat.RegistryFile)

	eb := &Winlogbeat{
		beat:   b,
		config: &config,
		done:   make(chan struct{}),
	}

	if err := eb.init(b); err != nil {
		return nil, err
	}

	return eb, nil
}

func (eb *Winlogbeat) init(b *beat.Beat) error {
	config := &eb.config.Winlogbeat

	// Create the event logs. This will validate the event log specific
	// configuration.
	eb.eventLogs = make([]eventlog.EventLog, 0, len(config.EventLogs))
	for _, config := range config.EventLogs {
		eventLog, err := eventlog.New(config)
		if err != nil {
			return fmt.Errorf("Failed to create new event log. %v", err)
		}
		debugf("Initialized EventLog[%s]", eventLog.Name())

		eb.eventLogs = append(eb.eventLogs, eventLog)
	}

	return nil
}

// Setup uses the loaded config and creates necessary markers and environment
// settings to allow the beat to be used.
func (eb *Winlogbeat) setup(b *beat.Beat) error {
	config := &eb.config.Winlogbeat

	eb.client = b.Publisher.Connect()

	var err error
	eb.checkpoint, err = checkpoint.NewCheckpoint(config.RegistryFile, 10, 5*time.Second)
	if err != nil {
		return err
	}

	if config.Metrics.BindAddress != "" {
		bindAddress := config.Metrics.BindAddress
		sock, err := net.Listen("tcp", bindAddress)
		if err != nil {
			return err
		}
		go func() {
			logp.Info("Metrics hosted at http://%s/debug/vars", bindAddress)
			err := http.Serve(sock, nil)
			if err != nil {
				logp.Warn("Unable to launch HTTP service for metrics. %v", err)
			}
		}()
	}

	return nil
}

// Run is used within the beats interface to execute the Winlogbeat workers.
func (eb *Winlogbeat) Run(b *beat.Beat) error {
	if err := eb.setup(b); err != nil {
		return err
	}

	persistedState := eb.checkpoint.States()

	// Initialize metrics.
	publishedEvents.Add("total", 0)
	ignoredEvents.Add("total", 0)

	var wg sync.WaitGroup
	for _, log := range eb.eventLogs {
		state, _ := persistedState[log.Name()]

		// Initialize per event log metrics.
		publishedEvents.Add(log.Name(), 0)
		ignoredEvents.Add(log.Name(), 0)

		// Start a goroutine for each event log.
		wg.Add(1)
		go eb.processEventLog(&wg, log, state)
	}

	wg.Wait()
	eb.checkpoint.Shutdown()
	return nil
}

// Stop is used to tell the winlogbeat that it should cease executing.
func (eb *Winlogbeat) Stop() {
	logp.Info("Stopping Winlogbeat")
	if eb.done != nil {
		eb.client.Close()
		close(eb.done)
	}
}

func (eb *Winlogbeat) processEventLog(
	wg *sync.WaitGroup,
	api eventlog.EventLog,
	state checkpoint.EventLogState,
) {
	defer wg.Done()

	err := api.Open(state.RecordNumber)
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
		case <-eb.done:
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

		events := make([]common.MapStr, 0, len(records))
		for _, lr := range records {
			events = append(events, lr.ToMapStr())
		}

		// Publish events.
		numEvents := int64(len(events))
		ok := eb.client.PublishEvents(events, publisher.Sync, publisher.Guaranteed)
		if !ok {
			// due to using Sync and Guaranteed the ok will only be false on shutdown.
			// Do not update the internal state and return in this case
			return
		}

		publishedEvents.Add("total", numEvents)
		publishedEvents.Add(api.Name(), numEvents)
		logp.Info("EventLog[%s] Successfully published %d events",
			api.Name(), numEvents)

		eb.checkpoint.Persist(api.Name(),
			records[len(records)-1].RecordID,
			records[len(records)-1].TimeCreated.SystemTime.UTC())
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
