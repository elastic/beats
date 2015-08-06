package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"time"

	"github.com/elastic/libbeat/cfgfile"
	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/publisher"
	"github.com/elastic/libbeat/service"
)

// You can overwrite these, e.g.: go build -ldflags "-X main.Version 1.0.0-beta3"
var Version = "1.0.0-beta2"
var Name = "topbeat"

type Topbeat struct {
	isAlive bool
	period  time.Duration
	procs   []string

	events chan common.MapStr
}

func (t *Topbeat) Init(config TopConfig, events chan common.MapStr) error {

	if config.Period != nil {
		t.period = time.Duration(*config.Period) * time.Second
	} else {
		t.period = 1 * time.Second
	}
	if config.Procs != nil {
		t.procs = *config.Procs
	} else {
		t.procs = []string{".*"} //all processes
	}

	logp.Debug("topbeat", "Init toppbeat")
	logp.Debug("topbeat", "Follow processes %q\n", t.procs)
	logp.Debug("topbeat", "Period %v\n", t.period)
	t.events = events
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

func (t *Topbeat) Run() error {

	t.isAlive = true

	for t.isAlive {
		time.Sleep(1 * time.Second)

		load_stat, err := GetSystemLoad()
		if err != nil {
			logp.Warn("Getting load statistics: %v", err)
			continue
		}
		cpu_stat, err := GetCpuTimes()
		if err != nil {
			logp.Warn("Getting cpu times: %v", err)
			continue
		}

		mem_stat, err := GetMemory()
		if err != nil {
			logp.Warn("Getting memory details: %v", err)
			continue
		}
		swap_stat, err := GetSwap()
		if err != nil {
			logp.Warn("Getting swap details: %v", err)
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

			if t.MatchProcess(process.Name) {

				logp.Debug("topbeat", "Process: %s", process)

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

		event := common.MapStr{
			"timestamp": common.Time(time.Now()),
			"type":      "system",
			"load":      load_stat,
			"cpu":       cpu_stat,
			"mem":       mem_stat,
			"swap":      swap_stat,
		}

		t.events <- event
	}
	return nil
}

func (t *Topbeat) Stop() {

	t.isAlive = false
}

func (t *Topbeat) IsAlive() bool {

	return t.isAlive
}

func main() {

	over := make(chan bool)

	// Use our own FlagSet, because some libraries pollute the global one
	var cmdLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	cfgfile.CmdLineFlags(cmdLine, Name)
	logp.CmdLineFlags(cmdLine)
	service.CmdLineFlags(cmdLine)

	publishDisabled := cmdLine.Bool("N", false, "Disable actual publishing for testing")
	printVersion := cmdLine.Bool("version", false, "Print version and exit")

	cmdLine.Parse(os.Args[1:])

	if *printVersion {
		fmt.Printf("%s version %s (%s)\n", Name, Version, runtime.GOARCH)
		return
	}

	err := cfgfile.Read(&Config)

	logp.Init(Name, &Config.Logging)

	logp.Debug("main", "Initializing output plugins")
	if err = publisher.Publisher.Init(*publishDisabled, Config.Output,
		Config.Shipper); err != nil {

		logp.Critical(err.Error())
		os.Exit(1)
	}

	topbeat := &Topbeat{}

	if err = topbeat.Init(Config.Input, publisher.Publisher.Queue); err != nil {
		logp.Critical(err.Error())
		os.Exit(1)
	}

	// Up to here was the initialization, now about running

	if cfgfile.IsTestConfig() {
		// all good, exit with 0
		os.Exit(0)
	}
	service.BeforeRun()

	// run the Beat code in background
	go func() {
		err := topbeat.Run()
		if err != nil {
			logp.Critical("Sniffer main loop failed: %v", err)
			os.Exit(1)
		}
		over <- true
	}()

	service.HandleSignals(topbeat.Stop)

	// Startup successful, disable stderr logging if requested by
	// cmdline flag
	logp.SetStderr()

	logp.Debug("main", "Starting topbeat")

	// Wait for the goroutines to finish
	for _ = range over {
		if !topbeat.IsAlive() {
			break
		}
	}

	logp.Debug("main", "Cleanup")
	service.Cleanup()
}
