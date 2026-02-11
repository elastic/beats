// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Borrowed from https://github.com/kolide/launcher/blob/master/cmd/osquery-extension/osquery-extension.go
// Original license from the kolide launcher repository

// MIT License

// Copyright (c) 2017 Kolide

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Expanded with Elastic custom extensions so we have only one binary to manager

package main

import (
	"flag"
	"os"
	"time"

	"github.com/osquery/osquery-go"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/client"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hooks"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/views"
)

var (
	socket   = flag.String("socket", "", "Path to the extensions UNIX domain socket")
	timeout  = flag.Int("timeout", 3, "Seconds to wait for autoloaded extensions")
	interval = flag.Int("interval", 3, "Seconds delay between connectivity checks")
	verbose  = flag.Bool("verbose", false, "Verbose logging")
)

func main() {
	flag.Parse()

	// Initialize glog-compatible logger with basic config
	// Will be reconfigured after connecting to osqueryd
	log := logger.New(os.Stderr, *verbose)

	// Hook manager for post hooks
	hooks := hooks.NewHookManager()

	if *socket == "" {
		log.Fatal("Missing required --socket argument")
	}

	timeoutD := time.Second * time.Duration(*timeout)

	// Wait for the socket to become available
	if err := waitForSocket(*socket, timeoutD, log); err != nil {
		log.Fatalf("Socket not available: %s", err)
	}

	// Create a client to query osqueryd configuration
	client, err := client.NewResilientClient(*socket, timeoutD, log)
	if err != nil {
		log.Warningf("Could not create resilient client: %s", err)
	} else {
		options, err := client.Options()
		if err != nil {
			log.Warningf("Could not retrieve osqueryd options: %s", err)
		} else {
			log.UpdateWithOsqueryOptions(options)
		}
	}

	serverTimeout := osquery.ServerTimeout(timeoutD)
	serverPingInterval := osquery.ServerPingInterval(
		time.Second * time.Duration(*interval),
	)

	go monitorForParent(log)

	server, err := osquery.NewExtensionManagerServer(
		"osquery-extension",
		*socket,
		serverTimeout,
		serverPingInterval,
	)
	if err != nil {
		log.Fatalf("Error creating extension: %s", err)
	}

	// Register the tables available for the specific platform build
	// Any module that needs to execute a post hook should register the hook
	// within this function
	RegisterTables(server, log, hooks, client)

	// Register tables and views generated from the specs
	tables.RegisterTables(server, log)
	views.RegisterViews(hooks, log)

	// Execute all post hooks to create any views required for the specific platform build
	go hooks.Execute(socket, log)

	if *verbose {
		log.Info("Starting osquery extension server")
	}

	if err := server.Run(); err != nil {
		log.Fatalf("Failed to run extension server: %s", err)
	}

	// Execute all shutdown hooks to clean up any resources
	hooks.Shutdown(socket, log)
}

// waitForSocket waits for the socket/pipe to become available
func waitForSocket(socketPath string, timeout time.Duration, log *logger.Logger) error {
	deadline := time.Now().Add(timeout)
	retryInterval := 500 * time.Millisecond
	for time.Now().Before(deadline) {
		log.Infof("Waiting for socket: %s", socketPath)
		if socketExists(socketPath) {
			log.Infof("Socket found: %s", socketPath)
			return nil
		}

		remaining := time.Until(deadline)
		if remaining < retryInterval {
			retryInterval = remaining
		}
		if retryInterval > 0 {
			time.Sleep(retryInterval)
		}
	}

	return os.ErrNotExist
}

// socketExists checks if a socket/pipe exists
func socketExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// continuously monitor for ppid and exit if osqueryd is no longer the parent process.
// because osqueryd is always the process starting the extension, when osqueryd is killed this process should also be cleaned up.
// sometimes the termination is not clean, causing this process to remain running, which sometimes prevents osqueryd from properly restarting.
// https://github.com/kolide/launcher/issues/341
func monitorForParent(log *logger.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	f := func() {
		ppid := os.Getppid()
		if ppid <= 1 {
			log.Error("extension process no longer owned by osqueryd, quitting")
			os.Exit(1)
		}
	}

	f()

	for range ticker.C {
		f()
	}
}
