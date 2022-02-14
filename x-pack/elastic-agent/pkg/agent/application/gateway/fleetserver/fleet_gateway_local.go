// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetserver

import (
	"context"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/gateway"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/pipeline"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi/client"
)

const gatewayWait = 2 * time.Second

var injectFleetServerInput = map[string]interface{}{
	// outputs is replaced by the fleet-server.spec
	"outputs": map[string]interface{}{
		"default": map[string]interface{}{
			"type":  "elasticsearch",
			"hosts": []string{"localhost:9200"},
		},
	},
	"inputs": []interface{}{
		map[string]interface{}{
			"type": "fleet-server",
		},
	},
}

// fleetServerWrapper wraps the fleetGateway to ensure that a local Fleet Server is running before trying
// to communicate with the gateway, which is local to the Elastic Agent.
type fleetServerWrapper struct {
	bgContext   context.Context
	log         *logger.Logger
	cfg         *configuration.FleetAgentConfig
	injectedCfg *config.Config
	wrapped     gateway.FleetGateway
	emitter     pipeline.EmitterFunc
}

// New creates a new fleet server gateway wrapping another fleet gateway.
func New(
	ctx context.Context,
	log *logger.Logger,
	cfg *configuration.FleetAgentConfig,
	rawConfig *config.Config,
	wrapped gateway.FleetGateway,
	emitter pipeline.EmitterFunc,
	injectServer bool) (gateway.FleetGateway, error) {
	if cfg.Server == nil || !injectServer {
		// not running a local Fleet Server
		return wrapped, nil
	}

	injectedCfg, err := injectFleetServer(rawConfig)
	if err != nil {
		return nil, errors.New(err, "failed to inject fleet-server input to start local Fleet Server", errors.TypeConfig)
	}

	return &fleetServerWrapper{
		bgContext:   ctx,
		log:         log,
		cfg:         cfg,
		injectedCfg: injectedCfg,
		wrapped:     wrapped,
		emitter:     emitter,
	}, nil
}

// Start starts the gateway.
func (w *fleetServerWrapper) Start() error {
	err := w.emitter(context.Background(), w.injectedCfg)
	if err != nil {
		return err
	}
	sleep(w.bgContext, gatewayWait)
	return w.wrapped.Start()
}

// SetClient sets the client for the wrapped gateway.
func (w *fleetServerWrapper) SetClient(c client.Sender) {
	w.wrapped.SetClient(c)
}

func injectFleetServer(rawConfig *config.Config) (*config.Config, error) {
	cfg := map[string]interface{}{}
	err := rawConfig.Unpack(cfg)
	if err != nil {
		return nil, err
	}
	cloned, err := config.NewConfigFrom(cfg)
	if err != nil {
		return nil, err
	}
	err = cloned.Merge(injectFleetServerInput)
	if err != nil {
		return nil, err
	}
	return cloned, nil
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
