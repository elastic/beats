// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package local

import (
	"bufio"
	"context"
	"os"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/x-pack/functionbeat/function/provider"
	"github.com/elastic/beats/v7/x-pack/functionbeat/function/telemetry"
	conf "github.com/elastic/elastic-agent-libs/config"
)

const stdinName = "stdin"

// Bundle exposes the local provider and the STDIN function.
var Bundle = provider.MustCreate(
	"local",
	provider.NewDefaultProvider("local", provider.NewNullCli, provider.NewNullTemplateBuilder),
	feature.MakeDetails("local events", "allows to trigger events locally.", feature.Experimental),
).MustAddFunction(
	stdinName,
	NewStdinFunction,
	feature.MakeDetails(stdinName, "read events from stdin", feature.Experimental),
).Bundle()

// StdinFunction reads events from STIN and terminates when stdin is completed.
type StdinFunction struct{}

// NewStdinFunction creates a new StdinFunction
func NewStdinFunction(
	provider provider.Provider,
	functionConfig *conf.C,
) (provider.Function, error) {
	return &StdinFunction{}, nil
}

// Run reads events from the STDIN and send them to the publisher pipeline, will stop reading by
// either by an external signal to stop or by reaching EOF. When EOF is reached functionbeat will shutdown.
func (s *StdinFunction) Run(ctx context.Context, client pipeline.ISyncClient, _ telemetry.T) error {
	errChan := make(chan error)
	defer close(errChan)
	lineChan := make(chan string)
	defer close(lineChan)

	// Make the os.Stdin interruptable, the shutdown cleanup will unblock the os.Stdin and the goroutine.
	go func(ctx context.Context, lineChan chan string, errChan chan error) {
		buf := bufio.NewReader(os.Stdin)
		scanner := bufio.NewScanner(buf)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				errChan <- err
				return
			}

			select {
			case <-ctx.Done():
				return
			case lineChan <- scanner.Text():
			}
		}
	}(ctx, lineChan, errChan)

	for {
		select {
		case <-ctx.Done():
			return os.Stdin.Close()
		case err := <-errChan:
			return err
		case line := <-lineChan:
			event := s.newEvent(line)
			err := client.Publish(event)
			if err != nil {
				return err
			}
		}
	}
}

func (s *StdinFunction) newEvent(line string) beat.Event {
	event := beat.Event{
		Timestamp: time.Now(),
		Fields: common.MapStr{
			"message": line,
		},
	}
	return event
}

// Name returns the name of the stdin function.
func (s *StdinFunction) Name() string {
	return stdinName
}
