// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	downloader "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download/localremote"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

const (
	agentName         = "elastic-agent"
	agentArtifactName = "beats/" + agentName
)

// After running Upgrade agent should download its own version specified by action
// from repository specified by fleet.
type handlerUpgrade struct {
	settings    *artifact.Config
	log         *logger.Logger
	emitter     emitterFunc
	dispatcher  programsDispatcher
	closers     []context.CancelFunc
	actionStore *actionStore
}

func (h *handlerUpgrade) Handle(ctx context.Context, a action, acker fleetAcker) error {
	h.log.Debugf("handlerUpgrade: action '%+v' received", a)
	action, ok := a.(*fleetapi.ActionUpgrade)
	if !ok {
		return fmt.Errorf("invalid type, expected ActionUpgrade and received %T", a)
	}

	// download artifact
	_, err := h.downloadArtifact(ctx, action)
	if err != nil {
		return err
	}

	// TODO: unpack correctly, skip root (symlink, config...) unpack data/*
	// TODO: change symlink
	// TODO: mark update happened so we can handle grace period
	// TODO: reexec
	return nil
}

func (h *handlerUpgrade) downloadArtifact(ctx context.Context, action *fleetapi.ActionUpgrade) (string, error) {
	settings := *h.settings
	if action.SourceURI != "" {
		settings.SourceURI = action.SourceURI
	}

	fetcher := downloader.NewDownloader(h.log, &settings)
	verifier, err := downloader.NewVerifier(h.log, &settings)
	if err != nil {
		return "", errors.New(err, "initiating verifier")
	}

	path, err := fetcher.Download(ctx, agentName, agentArtifactName, action.Version)
	if err != nil {
		return "", errors.New(err, "failed upgrade of agent binary")
	}

	matches, err := verifier.Verify(agentName, action.Version)
	if err != nil {
		return "", errors.New(err, "failed verification of agent binary")
	}
	if !matches {
		return "", errors.New("failed verification of agent binary, hash does not match", errors.TypeSecurity)
	}

	return path, nil
}
