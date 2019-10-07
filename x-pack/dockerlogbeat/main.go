// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"os"

	"github.com/docker/go-plugins-helpers/sdk"
	"github.com/elastic/beats/libbeat/common"
	logpcfg "github.com/elastic/beats/libbeat/logp/configure"
	_ "github.com/elastic/beats/libbeat/outputs/console"
	_ "github.com/elastic/beats/libbeat/outputs/elasticsearch"
	_ "github.com/elastic/beats/libbeat/outputs/fileout"
	_ "github.com/elastic/beats/libbeat/outputs/logstash"
	_ "github.com/elastic/beats/libbeat/publisher/queue/memqueue"
	_ "github.com/elastic/beats/libbeat/publisher/queue/spool"
	"github.com/elastic/beats/libbeat/service"
	"github.com/elastic/beats/x-pack/dockerlogbeat/pipelinemanager"
)

// genNewMonitoringConfig is a hacked-in function to enable a debug stderr logger
func genNewMonitoringConfig() (*common.Config, error) {
	cfgObject := make(map[string]string)
	cfgObject["level"] = "debug"
	cfgObject["to_stderr"] = "true"

	cfg, err := common.NewConfigFrom(cfgObject)
	if err != nil {
		return nil, err
	}
	return cfg, nil
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

	err = logpcfg.Logging("dockerbeat", logcfg)
	if err != nil {
		fatal("error starting log handler: %s", err)
	}

	pipelines := pipelinemanager.NewPipelineManager(logcfg)

	sdkHandler := sdk.NewHandler(`{"Implements": ["LoggingDriver"]}`)
	// Create handlers for startup and shutdown of the log driver
	sdkHandler.HandleFunc("/LogDriver.StartLogging", startLoggingHandler(pipelines))
	sdkHandler.HandleFunc("/LogDriver.StopLogging", stopLoggingHandler(pipelines))

	err = sdkHandler.ServeUnix("beatSocket", 0)
	if err != nil {
		fatal("Error in socket handler: %s", err)
	}
}
