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
	settings *artifact.Config
	log      *logger.Logger
	closers  []context.CancelFunc
}

func (h *handlerUpgrade) Handle(ctx context.Context, a action, acker fleetAcker) error {
	h.log.Debugf("handlerUpgrade: action '%+v' received", a)
	action, ok := a.(*fleetapi.ActionUpgrade)
	if !ok {
		return fmt.Errorf("invalid type, expected ActionUpgrade and received %T", a)
	}

	archivePath, err := h.downloadArtifact(ctx, action)
	if err != nil {
		return err
	}

	newHash, err := h.untar(ctx, action, archivePath)
	if err != nil {
		return err
	}

	if err := h.changeSymlink(ctx, action, newHash); err != nil {
		return err
	}

	if err := h.markUpgrade(ctx, action); err != nil {
		return err
	}

	return h.reexec(ctx, action)
}

func (h *handlerUpgrade) downloadArtifact(ctx context.Context, action *fleetapi.ActionUpgrade) (string, error) {
	// do not update source config
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

// untar unpacks archive correctly, skips root (symlink, config...) unpacks data/*
func (h *handlerUpgrade) untar(ctx context.Context, action *fleetapi.ActionUpgrade, archivePath string) (string, error) {
	return "", errors.New("not yet implemented")
}

// changeSymlink changes root symlink so it points to updated version
func (h *handlerUpgrade) changeSymlink(ctx context.Context, action *fleetapi.ActionUpgrade, newHash string) error {
	return errors.New("not yet implemented")
}

// markUpgrade marks update happened so we can handle grace period
func (h *handlerUpgrade) markUpgrade(ctx context.Context, action *fleetapi.ActionUpgrade) error {
	return errors.New("not yet implemented")
}

// reexec restarts agent so new version is run
func (h *handlerUpgrade) reexec(ctx context.Context, action *fleetapi.ActionUpgrade) error {
	return errors.New("not yet implemented")
}
