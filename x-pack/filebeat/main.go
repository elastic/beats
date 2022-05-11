// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/filebeat/cmd"
)

// The basic model of execution:
// - input: finds files in paths/globs to harvest, starts harvesters
// - harvester: reads a file, sends events to the spooler
// - spooler: buffers events until ready to flush to the publisher
// - publisher: writes to the network, notifies registrar
// - registrar: records positions of files read
// Finally, input uses the registrar information, on restart, to
// determine where in each file to restart a harvester.
func main() {
	port := fmt.Sprintf(":424%d", rand.Intn(10))
	go func() {
		s := &http.Server{
			Addr: port,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					fmt.Println("logging to stdout")
					fmt.Fprintf(os.Stderr, "logging to os.Stderr")
					fmt.Fprintln(w, "logged to stdout")
				case http.MethodPost:
					fmt.Fprintln(w, "panicking in 1 s")
					go func() {
						fmt.Println("logging to stdout before POST panic!")
						time.Sleep(time.Second)
						panic("HTTP PANIC!!")
					}()
				case "PANIC":
					fmt.Fprintln(w, "HTTP METHOD PANIC")
					go func() {
						fmt.Println("logging to stdout before PANIC panic!")
						time.Sleep(time.Second)
						panic("HTTP PANIC!")
					}()
				}
			}),
		}
		fmt.Printf("starting HTTP panic server on port %s\n", port)
		if err := s.ListenAndServe(); err != nil {
			logp.L().Error(fmt.Errorf("panic http server error: %w", err))
		}
	}()

	if err := cmd.Filebeat().Execute(); err != nil {
		os.Exit(1)
	}
}
