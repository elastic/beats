package beater

import (
	"errors"
	"math"
	"regexp"
	"strconv"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
	sigar "github.com/elastic/gosigar"
)

type ProcsMap map[int]*Process

type Topbeat struct {
	period           time.Duration
	procs            []string
	procsMap         ProcsMap
	lastCpuTimes     *CpuTimes
	lastCpuTimesList []CpuTimes
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

	t.procsMap = make(ProcsMap)

	if len(t.procs) == 0 {
		return
	}

	pids, err := Pids()
	if err != nil {
		logp.Warn("Getting the initial list of pids: %v", err)
	}

	for _, pid := range pids {
		process, err := GetProcess(pid, "")
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

	pids, err := Pids()
	if err != nil {
		logp.Warn("Getting the list of pids: %v", err)
		return err
	}

	newProcs := make(ProcsMap, len(pids))
	for _, pid := range pids {
		var cmdline string
		if previousProc := t.procsMap[pid]; previousProc != nil {
			cmdline = previousProc.CmdLine
		}

		process, err := GetProcess(pid, cmdline)
		if err != nil {
			logp.Debug("topbeat", "Skip process pid=%d: %v", pid, err)
			continue
		}

		if t.MatchProcess(process.Name) {

			newProcs[process.Pid] = process

			proc := common.MapStr{
				"pid":      process.Pid,
				"ppid":     process.Ppid,
				"name":     process.Name,
				"state":    process.State,
				"username": process.Username,
				"mem": common.MapStr{
					"size":  process.Mem.Size,
					"rss":   process.Mem.Resident,
					"rss_p": t.getProcMemPercentage(process, 0 /* read total mem usage */),
					"share": process.Mem.Share,
				},
				"cpu": common.MapStr{
					"user":       process.Cpu.User,
					"system":     process.Cpu.Sys,
					"total":      process.Cpu.Total,
					"total_p":    t.getProcCpuPercentage(process),
					"start_time": process.Cpu.FormatStartTime(),
				},
			}

			if process.CmdLine != "" {
				proc["cmdline"] = process.CmdLine
			}

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
	load_stat, err := GetSystemLoad()
	if err != nil {
		logp.Warn("Getting load statistics: %v", err)
		return err
	}
	cpu_stat, err := GetCpuTimes()
	if err != nil {
		logp.Warn("Getting cpu times: %v", err)
		return err
	}

	t.addCpuPercentage(cpu_stat)

	mem_stat, err := GetMemory()
	if err != nil {
		logp.Warn("Getting memory details: %v", err)
		return err
	}
	t.addMemPercentage(mem_stat)

	swap_stat, err := GetSwap()
	if err != nil {
		logp.Warn("Getting swap details: %v", err)
		return err
	}
	t.addSwapPercentage(swap_stat)

	event := common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"type":       "system",
		"load":       load_stat,
		"count":      1,
		"cpu": common.MapStr{
			"user":     cpu_stat.User,
			"system":   cpu_stat.Sys,
			"nice":     cpu_stat.Nice,
			"idle":     cpu_stat.Idle,
			"iowait":   cpu_stat.Wait,
			"irq":      cpu_stat.Irq,
			"softirq":  cpu_stat.SoftIrq,
			"steal":    cpu_stat.Stolen,
			"user_p":   cpu_stat.UserPercent,
			"system_p": cpu_stat.SystemPercent,
		},
		"mem": common.MapStr{
			"total":         mem_stat.Total,
			"used":          mem_stat.Used,
			"free":          mem_stat.Free,
			"actual_used":   mem_stat.ActualUsed,
			"actual_free":   mem_stat.ActualFree,
			"used_p":        mem_stat.UsedPercent,
			"actual_used_p": mem_stat.ActualUsedPercent,
		},
		"swap": common.MapStr{
			"total":  swap_stat.Total,
			"used":   swap_stat.Used,
			"free":   swap_stat.Free,
			"used_p": swap_stat.UsedPercent,
		},
	}

	if t.cpuPerCore {

		cpu_core_stat, err := GetCpuTimesList()
		if err != nil {
			logp.Warn("Getting cpu core times: %v", err)
			return err
		}
		t.addCpuPercentageList(cpu_core_stat)

		cpus := common.MapStr{}

		for coreNumber, stat := range cpu_core_stat {
			cpus["cpu"+strconv.Itoa(coreNumber)] = common.MapStr{
				"user":     stat.User,
				"system":   stat.Sys,
				"nice":     stat.Nice,
				"idle":     stat.Idle,
				"iowait":   stat.Wait,
				"irq":      stat.Irq,
				"softirq":  stat.SoftIrq,
				"steal":    stat.Stolen,
				"user_p":   stat.UserPercent,
				"system_p": stat.SystemPercent,
			}
		}
		event["cpus"] = cpus
	}

	t.events.PublishEvent(event)

	return nil
}

func (t *Topbeat) exportFileSystemStats() error {
	fss, err := GetFileSystemList()
	if err != nil {
		logp.Warn("Getting filesystem list: %v", err)
		return err
	}

	t.events.PublishEvents(collectFileSystemStats(fss))
	return nil
}

func collectFileSystemStats(fss []sigar.FileSystem) []common.MapStr {
	events := make([]common.MapStr, 0, len(fss))
	for _, fs := range fss {
		fsStat, err := GetFileSystemStat(fs)
		if err != nil {
			logp.Debug("topbeat", "Skip filesystem %d: %v", fsStat, err)
			continue
		}
		addFileSystemUsedPercentage(fsStat)

		event := common.MapStr{
			"@timestamp": common.Time(time.Now()),
			"type":       "filesystem",
			"count":      1,
			"fs": common.MapStr{
				"device_name": fsStat.DevName,
				"mount_point": fsStat.Mount,
				"total":       fsStat.Total,
				"used":        fsStat.Used,
				"free":        fsStat.Free,
				"avail":       fsStat.Avail,
				"files":       fsStat.Files,
				"free_files":  fsStat.FreeFiles,
				"used_p":      fsStat.UsedPercent,
			},
		}
		events = append(events, event)
	}
	return events
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

func (t *Topbeat) addMemPercentage(m *MemStat) {

	if m.Mem.Total == 0 {
		return
	}

	perc := float64(m.Mem.Used) / float64(m.Mem.Total)
	m.UsedPercent = Round(perc, .5, 2)

	actual_perc := float64(m.Mem.ActualUsed) / float64(m.Mem.Total)
	m.ActualUsedPercent = Round(actual_perc, .5, 2)
}

func (t *Topbeat) addSwapPercentage(s *SwapStat) {
	if s.Swap.Total == 0 {
		return
	}

	perc := float64(s.Swap.Used) / float64(s.Swap.Total)
	s.UsedPercent = Round(perc, .5, 2)
}

func addFileSystemUsedPercentage(f *FileSystemStat) {
	if f.Total == 0 {
		return
	}

	perc := float64(f.Used) / float64(f.Total)
	f.UsedPercent = Round(perc, .5, 2)
}

func (t *Topbeat) addCpuPercentage(t2 *CpuTimes) {

	t1 := t.lastCpuTimes

	if t1 != nil && t2 != nil {
		all_delta := t2.Cpu.Total() - t1.Cpu.Total()

		calculate := func(field2 uint64, field1 uint64) float64 {

			perc := 0.0
			delta := int64(field2 - field1)
			perc = float64(delta) / float64(all_delta)
			return Round(perc, .5, 4)
		}

		t2.UserPercent = calculate(t2.Cpu.User, t1.Cpu.User)
		t2.SystemPercent = calculate(t2.Cpu.Sys, t1.Cpu.Sys)
	}

	t.lastCpuTimes = t2

}

func (t *Topbeat) addCpuPercentageList(t2 []CpuTimes) {

	t1 := t.lastCpuTimesList

	if t1 != nil && t2 != nil && len(t1) == len(t2) {

		calculate := func(field2 uint64, field1 uint64, all_delta uint64) float64 {

			perc := 0.0
			delta := field2 - field1
			perc = float64(delta) / float64(all_delta)
			return Round(perc, .5, 4)
		}

		for i := 0; i < len(t1); i++ {
			all_delta := t2[i].Cpu.Total() - t1[i].Cpu.Total()
			t2[i].UserPercent = calculate(t2[i].Cpu.User, t1[i].Cpu.User, all_delta)
			t2[i].SystemPercent = calculate(t2[i].Cpu.Sys, t1[i].Cpu.Sys, all_delta)
		}

	}

	t.lastCpuTimesList = t2

}

func (t *Topbeat) getProcMemPercentage(proc *Process, total_phymem uint64) float64 {

	// in unit tests, total_phymem is set to a value greater than zero

	if total_phymem == 0 {
		mem_stat, err := GetMemory()
		if err != nil {
			logp.Warn("Getting memory details: %v", err)
			return 0
		}
		total_phymem = mem_stat.Mem.Total
	}

	perc := (float64(proc.Mem.Resident) / float64(total_phymem))

	return Round(perc, .5, 2)
}

func (t *Topbeat) getProcCpuPercentage(proc *Process) float64 {

	oproc, ok := t.procsMap[proc.Pid]
	if ok {

		delta_proc := (proc.Cpu.User - oproc.Cpu.User) + (proc.Cpu.Sys - oproc.Cpu.Sys)
		delta_time := proc.ctime.Sub(oproc.ctime).Nanoseconds() / 1e6 // in milliseconds
		perc := float64(delta_proc) / float64(delta_time)

		t.procsMap[proc.Pid] = proc

		return Round(perc, .5, 4)

	}
	return 0
}

func Round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}
