package beater

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/topbeat/system"
)

const (
	inputDeprecationWarning = "Using 'input' in configuration is deprecated " +
		"and is scheduled to be removed in Topbeat 6.0."
)

type Topbeat struct {
	period time.Duration

	TbConfig ConfigSettings
	events   publisher.Client

	sysStats  bool
	fsStats   bool
	coreStats bool

	cpu       *system.CPU
	procStats *system.ProcStats

	done chan struct{}
}

func New() *Topbeat {
	return &Topbeat{
		cpu:       &system.CPU{},
		procStats: &system.ProcStats{},
	}
}

func (tb *Topbeat) Config(b *beat.Beat) error {

	topbeatSection := "topbeat"
	if b.RawConfig.HasField("input") {
		// Copy input config to topbeat @deprecated
		logp.Warn(inputDeprecationWarning + " Use 'topbeat' instead.")
		topbeatSection = "input"

		if b.RawConfig.HasField("topbeat") {
			return fmt.Errorf("'topbeat' and 'input' are both set in config. Only " +
				"one can be enabled so use 'topbeat'. " + inputDeprecationWarning)
		}
	}

	rawTopbeatConfig, err := b.RawConfig.Child(topbeatSection, -1)
	if err != nil {
		logp.Err("Error reading configuration file: %v", err)
		return err
	}

	tb.TbConfig.Topbeat = defaultConfig
	err = rawTopbeatConfig.Unpack(&tb.TbConfig.Topbeat)
	if err != nil {
		logp.Err("Error reading configuration file: %v", err)
		return err
	}

	topbeatConfig := tb.TbConfig.Topbeat
	tb.period = topbeatConfig.Period
	tb.procStats.Procs = topbeatConfig.Procs
	tb.procStats.CpuTicks = topbeatConfig.Stats.CPUTicks
	tb.sysStats = topbeatConfig.Stats.System
	tb.procStats.ProcStats = topbeatConfig.Stats.Proc
	tb.fsStats = topbeatConfig.Stats.Filesystem
	tb.coreStats = topbeatConfig.Stats.Core
	tb.cpu.CpuTicks = topbeatConfig.Stats.CPUTicks

	logp.Debug("topbeat", "Init topbeat")
	logp.Debug("topbeat", "Follow processes %q", tb.procStats.Procs)
	logp.Debug("topbeat", "Period %v", tb.period)
	logp.Debug("topbeat", "System statistics %t", tb.sysStats)
	logp.Debug("topbeat", "Process statistics %t", tb.procStats.ProcStats)
	logp.Debug("topbeat", "File system statistics %t", tb.fsStats)
	logp.Debug("topbeat", "Export CPU usage for each core %t", tb.coreStats)
	logp.Debug("topbeat", "Export CPU usage in ticks %t", tb.cpu.CpuTicks)

	return nil
}

func (t *Topbeat) Setup(b *beat.Beat) error {
	t.events = b.Publisher.Connect()
	t.done = make(chan struct{})
	return nil
}

func (t *Topbeat) Run(b *beat.Beat) error {
	var err error

	err = t.procStats.InitProcStats()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(t.period)
	defer ticker.Stop()

	for {
		select {
		case <-t.done:
			return nil
		case <-ticker.C:
		}

		timerStart := time.Now()

		if t.sysStats {
			event, err := t.cpu.GetSystemStats()
			if err != nil {
				logp.Err("Error reading system stats: %v", err)
				break
			}
			t.events.PublishEvent(event)
		}
		if t.coreStats {
			events, err := t.cpu.GetCoreStats()
			if err != nil {
				logp.Err("Error reading per core stats: %v", err)
				break
			}
			t.events.PublishEvents(events)
		}
		if t.procStats.ProcStats {
			events, err := t.procStats.GetProcStatsEvents()
			if err != nil {
				logp.Err("Error reading proc stats: %v", err)
				break
			}
			t.events.PublishEvents(events)
		}
		if t.fsStats {
			events, err := system.GetFileSystemStats()
			if err != nil {
				logp.Err("Error reading fs stats: %v", err)
				break
			}
			t.events.PublishEvents(events)

		}

		timerEnd := time.Now()
		duration := timerEnd.Sub(timerStart)
		if duration.Nanoseconds() > t.period.Nanoseconds() {
			logp.Warn("Ignoring tick(s) due to processing taking longer than one period")
		}
	}

	return err
}

func (tb *Topbeat) Cleanup(b *beat.Beat) error {
	return nil
}

func (t *Topbeat) Stop() {
	logp.Info("Send stop signal to topbeat main loop")
	close(t.done)
	t.events.Close()
}
