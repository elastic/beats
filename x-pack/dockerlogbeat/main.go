// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/docker/go-plugins-helpers/sdk"

	"github.com/elastic/beats/v7/libbeat/common"
	logpcfg "github.com/elastic/beats/v7/libbeat/logp/configure"
	_ "github.com/elastic/beats/v7/libbeat/outputs/console"
	_ "github.com/elastic/beats/v7/libbeat/outputs/elasticsearch"
	_ "github.com/elastic/beats/v7/libbeat/outputs/fileout"
	_ "github.com/elastic/beats/v7/libbeat/outputs/kafka"
	_ "github.com/elastic/beats/v7/libbeat/outputs/logstash"
	_ "github.com/elastic/beats/v7/libbeat/outputs/redis"
	_ "github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	_ "github.com/elastic/beats/v7/libbeat/publisher/queue/spool"
	"github.com/elastic/beats/v7/libbeat/service"
	"github.com/elastic/beats/v7/x-pack/dockerlogbeat/pipelinemanager"
)

// genNewMonitoringConfig is a hacked-in function to enable a debug stderr logger
func genNewMonitoringConfig() (*common.Config, error) {
	lvl, isSet := os.LookupEnv("LOG_DRIVER_LEVEL")
	if !isSet {
		lvl = "info"
	}
	cfgObject := make(map[string]string)
	cfgObject["level"] = lvl
	cfgObject["to_stderr"] = "true"

	cfg, err := common.NewConfigFrom(cfgObject)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func setDestroyLogsOnStop() (bool, error) {
	setting, ok := os.LookupEnv("DESTROY_LOGS_ON_STOP")
	if !ok {
		return false, nil
	}
	return strconv.ParseBool(setting)
}

func fatal(format string, vs ...interface{}) {
	fmt.Fprintf(os.Stderr, format, vs...)
	os.Exit(1)
}

func main() {
	service.BeforeRun()
	defer service.Cleanup()

	logcfg, err := genNewMonitoringConfig()
	if err != nil {
		fatal("error starting config: %s", err)
	}

	err = logpcfg.Logging("elastic-logging-driver", logcfg)
	if err != nil {
		fatal("error starting log handler: %s", err)
	}

	logDestroy, err := setDestroyLogsOnStop()
	if err != nil {
		fatal("DESTROY_LOGS_ON_STOP must be 'true' or 'false': %s", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		fatal("Error fetching hostname: %s", err)
	}

	pipelines := pipelinemanager.NewPipelineManager(logDestroy, hostname)

	sdkHandler := sdk.NewHandler(`{"Implements": ["LoggingDriver"]}`)
	// Create handlers for startup and shutdown of the log driver
	sdkHandler.HandleFunc("/LogDriver.StartLogging", startLoggingHandler(pipelines))
	sdkHandler.HandleFunc("/LogDriver.StopLogging", stopLoggingHandler(pipelines))
	sdkHandler.HandleFunc("/LogDriver.Capabilities", reportCaps())
	sdkHandler.HandleFunc("/LogDriver.ReadLogs", readLogHandler(pipelines))

	err = sdkHandler.ServeUnix("beatSocket", 0)
	if err != nil {
		fatal("Error in socket handler: %s", err)
	}
}
