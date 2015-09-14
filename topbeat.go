package main

import (
	"math"
	"regexp"
	"time"

	"github.com/elastic/libbeat/beat"
	"github.com/elastic/libbeat/cfgfile"
	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/publisher"
)

type ProcsMap map[int]*Process

type Topbeat struct {
	isAlive      bool
	period       time.Duration
	procs        []string
	procsMap     ProcsMap
	lastCpuTimes *CpuTimes
	TbConfig     ConfigSettings
	events       chan common.MapStr
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
		tb.period = 1 * time.Second
	}
	if tb.TbConfig.Input.Procs != nil {
		tb.procs = *tb.TbConfig.Input.Procs
	} else {
		tb.procs = []string{".*"} //all processes
	}

	logp.Debug("topbeat", "Init toppbeat")
	logp.Debug("topbeat", "Follow processes %q\n", tb.procs)
	logp.Debug("topbeat", "Period %v\n", tb.period)

	return nil
}

func (tb *Topbeat) Setup(b *beat.Beat) error {

	tb.events = publisher.Publisher.Queue
	return nil
}

func (t *Topbeat) Run(b *beat.Beat) error {

	t.isAlive = true

	t.initProcStats()

	var err error

	for t.isAlive {
		time.Sleep(t.period)

		err = t.exportSystemStats()
		if err != nil {
			logp.Err("Error reading system stats: %v", err)
		}
		err = t.exportProcStats()
		if err != nil {
			logp.Err("Error reading proc stats: %v", err)
		}
		err = t.exportFileSystemStats()
		if err != nil {
			logp.Err("Error reading fs stats: %v", err)
		}
	}

	return err
}

func (tb *Topbeat) Cleanup(b *beat.Beat) error {
	return nil
}

func (t *Topbeat) Stop() {

	t.isAlive = false
}

func (t *Topbeat) initProcStats() {

	t.procsMap = make(ProcsMap)

	if len(t.procs) == 0 {
		return
	}

	pids, err := Pids()
	if err != nil {
		logp.Warn("Getting the list of pids: %v", err)
	}

	for _, pid := range pids {
		process, err := GetProcess(pid)
		if err != nil {
			logp.Debug("topbeat", "Skip process %d: %v", pid, err)
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

	for _, pid := range pids {
		process, err := GetProcess(pid)
		if err != nil {
			logp.Debug("topbeat", "Skip process %d: %v", pid, err)
			continue
		}

		if t.MatchProcess(process.Name) {

			t.addProcCpuPercentage(process)
			t.addProcMemPercentage(process, 0 /*read total mem usage */)

			t.procsMap[process.Pid] = process

			event := common.MapStr{
				"timestamp":  common.Time(time.Now()),
				"type":       "proc",
				"proc.pid":   process.Pid,
				"proc.ppid":  process.Ppid,
				"proc.name":  process.Name,
				"proc.state": process.State,
				"proc.mem":   process.Mem,
				"proc.cpu":   process.Cpu,
			}
			t.events <- event
		}
	}
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
	t.addMemPercentage(swap_stat)

	event := common.MapStr{
		"timestamp": common.Time(time.Now()),
		"type":      "system",
		"load":      load_stat,
		"cpu":       cpu_stat,
		"mem":       mem_stat,
		"swap":      swap_stat,
	}

	t.events <- event

	return nil
}

func (t *Topbeat) exportFileSystemStats() error {

	fss, err := GetFileSystemList()
	if err != nil {
		logp.Warn("Getting filesystem list: %v", err)
		return err
	}

	for _, fs := range fss {
		fs_stat, err := GetFileSystemStat(fs)
		if err != nil {
			logp.Debug("topbeat", "Skip filesystem %d: %v", fs_stat, err)
			continue
		}
		t.addFileSystemUsedPercentage(fs_stat)

		event := common.MapStr{
			"timestamp": common.Time(time.Now()),
			"type":      "filesystem",
			"fs":        fs_stat,
		}
		t.events <- event
	}

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

func (t *Topbeat) addMemPercentage(m *MemStat) {

	if m.Total == 0 {
		return
	}

	perc := float64(m.Used) / float64(m.Total)
	m.UsedPercent = Round(perc, .5, 2)
}

func (t *Topbeat) addFileSystemUsedPercentage(f *FileSystemStat) {

	if f.Total == 0 {
		return
	}

	perc := float64(f.Used) / float64(f.Total)
	f.UsedPercent = Round(perc, .5, 2)
}

func (t *Topbeat) addCpuPercentage(t2 *CpuTimes) {

	t1 := t.lastCpuTimes

	if t1 != nil && t2 != nil {
		all_delta := t2.sum() - t1.sum()

		calculate := func(field2 uint64, field1 uint64) float64 {

			perc := 0.0
			delta := field2 - field1
			perc = float64(delta) / float64(all_delta)
			return Round(perc, .5, 2)
		}

		t2.UserPercent = calculate(t2.User, t1.User)
		t2.SystemPercent = calculate(t2.System, t1.System)
	}

	t.lastCpuTimes = t2

}

func (t *Topbeat) addProcMemPercentage(proc *Process, total_phymem uint64) {

	// in unit tests, total_phymem is set to a value greater than zero

	if total_phymem == 0 {
		mem_stat, err := GetMemory()
		if err != nil {
			logp.Warn("Getting memory details: %v", err)
			return
		}
		total_phymem = mem_stat.Total
	}

	perc := (float64(proc.Mem.Rss) / float64(total_phymem))

	proc.Mem.RssPercent = Round(perc, .5, 2)
}

func (t *Topbeat) addProcCpuPercentage(proc *Process) {

	oproc, ok := t.procsMap[proc.Pid]
	if ok {

		delta_proc := (proc.Cpu.User - oproc.Cpu.User) + (proc.Cpu.System - oproc.Cpu.System)
		delta_time := proc.ctime.Sub(oproc.ctime).Nanoseconds() / 1e6 // in milliseconds
		perc := float64(delta_proc) / float64(delta_time)

		t.procsMap[proc.Pid] = proc

		proc.Cpu.UserPercent = Round(perc, .5, 2)

	}
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

func (t *CpuTimes) sum() uint64 {
	return t.User + t.Nice + t.System + t.Idle + t.IOWait + t.Irq + t.SoftIrq + t.Steal
}
