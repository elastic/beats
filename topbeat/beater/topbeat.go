package beater

import (
	"errors"
	"regexp"
	"strconv"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/topbeat/system"
)

type Topbeat struct {
	period           time.Duration
	procs            []string
	procsMap         system.ProcsMap
	lastCpuTimes     *system.CpuTimes
	lastCpuTimesList []system.CpuTimes
	TbConfig         ConfigSettings
	events           publisher.Client

	sysStats   bool
	procStats  bool
	fsStats    bool
	cpuPerCore bool

	done chan struct{}
}

func New() *Topbeat {
	return &Topbeat{}
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
		tb.procs = *tb.TbConfig.Input.Procs
	} else {
		tb.procs = []string{".*"} //all processes
	}

	if tb.TbConfig.Input.Stats.System != nil {
		tb.sysStats = *tb.TbConfig.Input.Stats.System
	} else {
		tb.sysStats = true
	}
	if tb.TbConfig.Input.Stats.Proc != nil {
		tb.procStats = *tb.TbConfig.Input.Stats.Proc
	} else {
		tb.procStats = true
	}
	if tb.TbConfig.Input.Stats.Filesystem != nil {
		tb.fsStats = *tb.TbConfig.Input.Stats.Filesystem
	} else {
		tb.fsStats = true
	}
	if tb.TbConfig.Input.Stats.CpuPerCore != nil {
		tb.cpuPerCore = *tb.TbConfig.Input.Stats.CpuPerCore
	} else {
		tb.cpuPerCore = false
	}

	if !tb.sysStats && !tb.procStats && !tb.fsStats {
		return errors.New("Invalid statistics configuration")
	}

	logp.Debug("topbeat", "Init topbeat")
	logp.Debug("topbeat", "Follow processes %q\n", tb.procs)
	logp.Debug("topbeat", "Period %v\n", tb.period)
	logp.Debug("topbeat", "System statistics %t\n", tb.sysStats)
	logp.Debug("topbeat", "Process statistics %t\n", tb.procStats)
	logp.Debug("topbeat", "File system statistics %t\n", tb.fsStats)
	logp.Debug("topbeat", "Cpu usage per core %t\n", tb.cpuPerCore)

	return nil
}

func (tb *Topbeat) Setup(b *beat.Beat) error {
	tb.events = b.Events
	tb.done = make(chan struct{})
	return nil
}

func (t *Topbeat) Run(b *beat.Beat) error {
	var err error

	t.initProcStats()

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
			err = t.exportSystemStats()
			if err != nil {
				logp.Err("Error reading system stats: %v", err)
				break
			}
		}
		if t.procStats {
			err = t.exportProcStats()
			if err != nil {
				logp.Err("Error reading proc stats: %v", err)
				break
			}
		}
		if t.fsStats {
			err = t.exportFileSystemStats()
			if err != nil {
				logp.Err("Error reading fs stats: %v", err)
				break
			}
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

func (t *Topbeat) initProcStats() {

	t.procsMap = make(system.ProcsMap)

	if len(t.procs) == 0 {
		return
	}

	pids, err := system.Pids()
	if err != nil {
		logp.Warn("Getting the initial list of pids: %v", err)
	}

	for _, pid := range pids {
		process, err := system.GetProcess(pid, "")
		if err != nil {
			logp.Debug("topbeat", "Skip process pid=%d: %v", pid, err)
			continue
		}
		t.procsMap[process.Pid] = process
	}
}

func (t *Topbeat) exportProcStats() error {

	if len(t.procs) == 0 {
		return nil
	}

	pids, err := system.Pids()
	if err != nil {
		logp.Warn("Getting the list of pids: %v", err)
		return err
	}

	newProcs := make(system.ProcsMap, len(pids))
	for _, pid := range pids {
		var cmdline string
		if previousProc := t.procsMap[pid]; previousProc != nil {
			cmdline = previousProc.CmdLine
		}

		process, err := system.GetProcess(pid, cmdline)
		if err != nil {
			logp.Debug("topbeat", "Skip process pid=%d: %v", pid, err)
			continue
		}

		if t.MatchProcess(process.Name) {

			newProcs[process.Pid] = process

			last, ok := t.procsMap[process.Pid]
			if ok {
				t.procsMap[process.Pid] = process
			}
			proc := system.GetProcessEvent(process, last)

			event := common.MapStr{
				"@timestamp": common.Time(time.Now()),
				"type":       "process",
				"count":      1,
				"proc":       proc,
			}

			t.events.PublishEvent(event)
		}
	}
	t.procsMap = newProcs
	return nil
}

func (t *Topbeat) exportSystemStats() error {
	load_stat, err := system.GetSystemLoad()
	if err != nil {
		logp.Warn("Getting load statistics: %v", err)
		return err
	}
	cpuStat, err := system.GetCpuTimes()
	if err != nil {
		logp.Warn("Getting cpu times: %v", err)
		return err
	}

	t.addCpuPercentage(cpuStat)

	memStat, err := system.GetMemory()
	if err != nil {
		logp.Warn("Getting memory details: %v", err)
		return err
	}
	system.AddMemPercentage(memStat)

	swapStat, err := system.GetSwap()
	if err != nil {
		logp.Warn("Getting swap details: %v", err)
		return err
	}
	system.AddSwapPercentage(swapStat)

	event := common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"type":       "system",
		"load":       load_stat,
		"count":      1,
		"cpu":        system.GetCpuStatEvent(cpuStat),
		"mem":        system.GetMemoryEvent(memStat),
		"swap":       system.GetSwapEvent(swapStat),
	}

	if t.cpuPerCore {

		cpuCoreStat, err := system.GetCpuTimesList()
		if err != nil {
			logp.Warn("Getting cpu core times: %v", err)
			return err
		}
		t.addCpuPercentageList(cpuCoreStat)

		cpus := common.MapStr{}

		for coreNumber, stat := range cpuCoreStat {
			cpus["cpu"+strconv.Itoa(coreNumber)] = system.GetCpuStatEvent(&stat)
		}
		event["cpus"] = cpus
	}

	t.events.PublishEvent(event)

	return nil
}

func (t *Topbeat) exportFileSystemStats() error {
	fss, err := system.GetFileSystemList()
	if err != nil {
		logp.Warn("Getting filesystem list: %v", err)
		return err
	}

	t.events.PublishEvents(system.CollectFileSystemStats(fss))
	return nil
}

func (t *Topbeat) MatchProcess(name string) bool {

	for _, reg := range t.procs {
		matched, _ := regexp.MatchString(reg, name)
		if matched {
			return true
		}
	}
	return false
}

func (t *Topbeat) addCpuPercentage(t2 *system.CpuTimes) {
	t.lastCpuTimes = system.GetCpuPercentage(t.lastCpuTimes, t2)
}

func (t *Topbeat) addCpuPercentageList(t2 []system.CpuTimes) {
	t.lastCpuTimesList = system.GetCpuPercentageList(t.lastCpuTimesList, t2)
}
