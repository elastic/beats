package beater

import (
	"errors"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/topbeat/system"
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

	err := cfgfile.Read(&tb.TbConfig, "")
	if err != nil {
		logp.Err("Error reading configuration file: %v", err)
		return err
	}

	if tb.TbConfig.Input.Period != nil {
		tb.period = time.Duration(*tb.TbConfig.Input.Period) * time.Second
	} else {
		tb.period = 10 * time.Second
	}
	if tb.TbConfig.Input.Procs != nil {
		tb.procStats.Procs = *tb.TbConfig.Input.Procs
	} else {
		tb.procStats.Procs = []string{".*"} //all processes
	}

	if tb.TbConfig.Input.Stats.System != nil {
		tb.sysStats = *tb.TbConfig.Input.Stats.System
	} else {
		tb.sysStats = true
	}
	if tb.TbConfig.Input.Stats.Proc != nil {
		tb.procStats.ProcStats = *tb.TbConfig.Input.Stats.Proc
	} else {
		tb.procStats.ProcStats = true
	}
	if tb.TbConfig.Input.Stats.Filesystem != nil {
		tb.fsStats = *tb.TbConfig.Input.Stats.Filesystem
	} else {
		tb.fsStats = true
	}
	if tb.TbConfig.Input.Stats.CpuPerCore != nil {
		tb.cpu.CpuPerCore = *tb.TbConfig.Input.Stats.CpuPerCore
	} else {
		tb.cpu.CpuPerCore = false
	}

	if !tb.sysStats && !tb.procStats.ProcStats && !tb.fsStats {
		return errors.New("Invalid statistics configuration")
	}

	logp.Debug("topbeat", "Init topbeat")
	logp.Debug("topbeat", "Follow processes %q\n", tb.procStats.Procs)
	logp.Debug("topbeat", "Period %v\n", tb.period)
	logp.Debug("topbeat", "System statistics %t\n", tb.sysStats)
	logp.Debug("topbeat", "Process statistics %t\n", tb.procStats.ProcStats)
	logp.Debug("topbeat", "File system statistics %t\n", tb.fsStats)
	logp.Debug("topbeat", "Cpu usage per core %t\n", tb.cpu.CpuPerCore)

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
