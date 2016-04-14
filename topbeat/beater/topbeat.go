package beater

import (
	"errors"
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

	sysStats bool
	fsStats  bool

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

	err := b.RawConfig.Unpack(&tb.TbConfig)
	if err != nil {
		logp.Err("Error reading configuration file: %v", err)
		return err
	}

	if tb.TbConfig.Topbeat != nil && tb.TbConfig.Input != nil {
		return fmt.Errorf("'topbeat' and 'input' are both set in config. Only " +
			"one can be enabled so use 'topbeat'. " + inputDeprecationWarning)
	}

	// Copy input config to topbeat @deprecated
	if tb.TbConfig.Input != nil {
		logp.Warn(inputDeprecationWarning + " Use 'topbeat' instead.")
		tb.TbConfig.Topbeat = tb.TbConfig.Input
	}

	topbeatConfig := tb.TbConfig.Topbeat

	if topbeatConfig.Period != nil {
		tb.period = time.Duration(*topbeatConfig.Period) * time.Second
	} else {
		tb.period = 10 * time.Second
	}
	if topbeatConfig.Procs != nil {
		tb.procStats.Procs = *topbeatConfig.Procs
	} else {
		tb.procStats.Procs = []string{".*"} //all processes
	}

	if topbeatConfig.Stats.System != nil {
		tb.sysStats = *topbeatConfig.Stats.System
	} else {
		tb.sysStats = true
	}
	if topbeatConfig.Stats.Proc != nil {
		tb.procStats.ProcStats = *topbeatConfig.Stats.Proc
	} else {
		tb.procStats.ProcStats = true
	}
	if topbeatConfig.Stats.Filesystem != nil {
		tb.fsStats = *topbeatConfig.Stats.Filesystem
	} else {
		tb.fsStats = true
	}
	if topbeatConfig.Stats.CpuPerCore != nil {
		tb.cpu.CpuPerCore = *topbeatConfig.Stats.CpuPerCore
	} else {
		tb.cpu.CpuPerCore = false
	}

	if !tb.sysStats && !tb.procStats.ProcStats && !tb.fsStats {
		return errors.New("Invalid statistics configuration")
	}

	logp.Debug("topbeat", "Init topbeat")
	logp.Debug("topbeat", "Follow processes %q", tb.procStats.Procs)
	logp.Debug("topbeat", "Period %v", tb.period)
	logp.Debug("topbeat", "System statistics %t", tb.sysStats)
	logp.Debug("topbeat", "Process statistics %t", tb.procStats.ProcStats)
	logp.Debug("topbeat", "File system statistics %t", tb.fsStats)
	logp.Debug("topbeat", "Cpu usage per core %t", tb.cpu.CpuPerCore)

	return nil
}

func (tb *Topbeat) Setup(b *beat.Beat) error {
	tb.events = b.Events
	tb.done = make(chan struct{})
	return nil
}

func (t *Topbeat) Run(b *beat.Beat) error {
	var err error

	t.procStats.InitProcStats()

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
		if t.procStats.ProcStats {
			events, err := t.procStats.GetProcStats()
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
	close(t.done)
}
