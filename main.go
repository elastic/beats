package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/elastic/libbeat/cfgfile"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/publisher"
	"github.com/elastic/libbeat/service"
)

// You can overwrite these, e.g.: go build -ldflags "-X main.Version 1.0.0-beta3"
var Version = "1.0.0-beta2"
var Name = "topbeat"

func main() {

	// Use our own FlagSet, because some libraries pollute the global one
	var cmdLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	cfgfile.CmdLineFlags(cmdLine, Name)
	logp.CmdLineFlags(cmdLine)
	service.CmdLineFlags(cmdLine)
	publisher.CmdLineFlags(cmdLine)

	printVersion := cmdLine.Bool("version", false, "Print version and exit")

	cmdLine.Parse(os.Args[1:])

	if *printVersion {
		fmt.Printf("%s version %s (%s)\n", Name, Version, runtime.GOARCH)
		return
	}

	err := cfgfile.Read(&Config)

	logp.Init(Name, &Config.Logging)

	logp.Debug("main", "Initializing output plugins")
	if err = publisher.Publisher.Init(Name, Config.Output, Config.Shipper); err != nil {
		logp.Critical(err.Error())
		os.Exit(1)
	}

	topbeat := &Topbeat{}
	if err = topbeat.Init(Config.Input, publisher.Publisher.Queue); err != nil {
		logp.Critical(err.Error())
		os.Exit(1)
	}

	// Up to here was the initialization
	if cfgfile.IsTestConfig() {
		// all good, exit with 0
		os.Exit(0)
	}
	service.BeforeRun()

	service.HandleSignals(topbeat.Stop)

	// Startup successful, disable stderr logging if requested by
	// cmdline flag
	logp.SetStderr()

	logp.Debug("main", "Starting topbeat")

	err = topbeat.Run()
	if err != nil {
		logp.Critical("Sniffer main loop failed: %v", err)
		os.Exit(1)
	}

	logp.Debug("main", "Cleanup")
	service.Cleanup()
}
