// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package kprobeprovider

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/processdb"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/provider"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/types"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	tracingEvents = "/sys/kernel/debug/tracing/kprobe_events"
	tracingPipe   = "/sys/kernel/tracing/trace_pipe"

	loadExecve = "p:kprobes/my_probe sys_execve\n"
)

type prvdr struct {
	ctx    context.Context
	logger *logp.Logger
	db     *processdb.DB
}

func NewProvider(ctx context.Context, logger *logp.Logger, db *processdb.DB) (provider.Provider, error) {
	p := prvdr{
		ctx:    ctx,
		logger: logger,
		db:     db,
	}

	// Load kprobe
	eventsFile, err := os.OpenFile(tracingEvents, os.O_APPEND|os.O_RDWR, 0777)
	if err != nil {
		return nil, fmt.Errorf("opening %v: %w", tracingEvents, err)
	}
	defer eventsFile.Close()

	if _, err := eventsFile.WriteString(loadExecve); err != nil {
		return nil, fmt.Errorf("loading execve kprobe: %w", err)
	}

	pipeFile, err := os.OpenFile(tracingPipe, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening trace pipe: %w", err)
	}
	//Read from trace pipe, and insert events into DB.
	go func(f *os.File, logger *logp.Logger) {
		reader := bufio.NewReader(f)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				logger.Errorf("reading event pipe: %v", err)
			}
			logger.Errorf("MWOLF: pipe: %v", line)
		}
	}(pipeFile, logger)
	return &p, nil
}

func (p prvdr) GetProcess(pid uint32) (*types.Process, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p prvdr) SyncDB(ev *beat.Event, pid uint32) error {
	return nil
}
