// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux && !((amd64 || arm64) && cgo)

package kerneltracingprovider

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v9/libbeat/beat"
	"github.com/elastic/beats/v9/x-pack/auditbeat/processors/sessionmd/provider"
	"github.com/elastic/beats/v9/x-pack/auditbeat/processors/sessionmd/types"
	"github.com/elastic/elastic-agent-libs/logp"
)

type prvdr struct{}

func NewProvider(ctx context.Context, logger *logp.Logger) (provider.Provider, error) {
	return prvdr{}, fmt.Errorf("build type not supported, cgo required")
}

func (p prvdr) Sync(event *beat.Event, pid uint32) error {
	return fmt.Errorf("build type not supported")
}

func (p prvdr) GetProcess(pid uint32) (*types.Process, error) {
	return nil, fmt.Errorf("build type not supported")
}
