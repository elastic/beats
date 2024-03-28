// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/paths"
)

// Input defines an interface for kvstore-based inputs.
type Input interface {
	// Name reports the input name.
	Name() string

	// Test runs the Test method for the configured source.
	Test(testCtx v2.TestContext) error

	// Run starts the data collection. Run must return an error only if the
	// error is fatal, making it impossible for the input to recover.
	Run(inputCtx v2.Context, store *Store, client beat.Client) error
}

// input implements the v2.Input interface.
var _ v2.Input = &input{}

type input struct {
	id           string
	manager      *Manager
	managedInput Input
}

// Name returns the name of this input.
func (n *input) Name() string {
	return n.managedInput.Name()
}

// Test runs the Test method for the managed input.
func (n *input) Test(testCtx v2.TestContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("input %s test panic with: %+v\n%s", n.Name(), r, debug.Stack())
			testCtx.Logger.Errorf("Input %s test panic: %+v", n.Name(), err)
		}
	}()

	return n.managedInput.Test(testCtx)
}

// Run runs data collection for the managed input. If a panic occurs, we create
// an error value with stack trace to report the issue, but not crash the whole process.
func (n *input) Run(runCtx v2.Context, connector beat.PipelineConnector) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("input %s panic with: %+v\n%s", runCtx.ID, r, debug.Stack())
			runCtx.Logger.Errorf("Input %s panic: %+v", runCtx.ID, err)
		}
	}()

	client, err := connector.ConnectWith(beat.ClientConfig{
		EventListener: NewTxACKHandler(),
	})
	if err != nil {
		return fmt.Errorf("could not connect to publishing pipeline: %s", err)
	}
	defer client.Close()

	dataDir := paths.Resolve(paths.Data, "kvstore")
	if err = os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("kvstore: unable to make data directory: %w", err)
	}
	filename := filepath.Join(dataDir, runCtx.ID+".db")
	store, err := NewStore(runCtx.Logger, filename, 0600)
	if err != nil {
		return err
	}
	defer store.Close()

	return n.managedInput.Run(runCtx, store, client)
}
