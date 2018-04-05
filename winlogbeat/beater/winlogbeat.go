/*
Package beater provides the implementation of the libbeat Beater interface for
Winlogbeat. The main event loop is implemented in this package.
*/
package beater

import (
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"

	"github.com/elastic/beats/winlogbeat/checkpoint"
	"github.com/elastic/beats/winlogbeat/config"
	"github.com/elastic/beats/winlogbeat/eventlog"
)

// Debug logging functions for this package.
var (
	debugf = logp.MakeDebug("winlogbeat")
)

// Time the application was started.
var startTime = time.Now().UTC()

// Winlogbeat is used to conform to the beat interface
type Winlogbeat struct {
	beat       *beat.Beat              // Common beat information.
	config     config.WinlogbeatConfig // Configuration settings.
	eventLogs  []*eventLogger          // List of all event logs being monitored.
	done       chan struct{}           // Channel to initiate shutdown of main event loop.
	pipeline   beat.Pipeline           // Interface to publish event.
	checkpoint *checkpoint.Checkpoint  // Persists event log state to disk.
}

// New returns a new Winlogbeat.
func New(b *beat.Beat, _ *common.Config) (beat.Beater, error) {
	// Read configuration.
	config := config.DefaultSettings
	err := b.BeatConfig.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("Error reading configuration file. %v", err)
	}

	// resolve registry file path
	config.RegistryFile = paths.Resolve(paths.Data, config.RegistryFile)
	logp.Info("State will be read from and persisted to %s",
		config.RegistryFile)

	eb := &Winlogbeat{
		beat:   b,
		config: config,
		done:   make(chan struct{}),
	}

	if err := eb.init(b); err != nil {
		return nil, err
	}

	return eb, nil
}

func (eb *Winlogbeat) init(b *beat.Beat) error {
	config := &eb.config

	// Create the event logs. This will validate the event log specific
	// configuration.
	eb.eventLogs = make([]*eventLogger, 0, len(config.EventLogs))
	for _, config := range config.EventLogs {
		eventLog, err := eventlog.New(config)
		if err != nil {
			return fmt.Errorf("Failed to create new event log. %v", err)
		}
		debugf("Initialized EventLog[%s]", eventLog.Name())

		logger, err := newEventLogger(eventLog, config)
		if err != nil {
			return fmt.Errorf("Failed to create new event log. %v", err)
		}

		eb.eventLogs = append(eb.eventLogs, logger)
	}

	return nil
}

// Setup uses the loaded config and creates necessary markers and environment
// settings to allow the beat to be used.
func (eb *Winlogbeat) setup(b *beat.Beat) error {
	config := &eb.config

	var err error
	eb.checkpoint, err = checkpoint.NewCheckpoint(config.RegistryFile, 10, 5*time.Second)
	if err != nil {
		return err
	}

	eb.pipeline = b.Publisher
	return nil
}

// Run is used within the beats interface to execute the Winlogbeat workers.
func (eb *Winlogbeat) Run(b *beat.Beat) error {
	if err := eb.setup(b); err != nil {
		return err
	}

	persistedState := eb.checkpoint.States()

	// Initialize metrics.
	initMetrics("total")

	// setup global event ACK handler
	err := eb.pipeline.SetACKHandler(beat.PipelineACKHandler{
		ACKLastEvents: func(data []interface{}) {
			for _, datum := range data {
				if st, ok := datum.(checkpoint.EventLogState); ok {
					eb.checkpoint.PersistState(st)
				}
			}
		},
	})
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, log := range eb.eventLogs {
		state, _ := persistedState[log.source.Name()]

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
		close(eb.done)
	}
}

func (eb *Winlogbeat) processEventLog(
	wg *sync.WaitGroup,
	logger *eventLogger,
	state checkpoint.EventLogState,
) {
	defer wg.Done()
	logger.run(eb.done, eb.pipeline, state)
}
