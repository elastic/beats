// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package upgrade

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

const (
	agentName         = "elastic-agent"
	hashLen           = 6
	agentCommitFile   = ".elastic-agent.active.commit"
	agentArtifactName = "beats/" + agentName
)

// Upgrader performs an upgrade
type Upgrader struct {
	settings *artifact.Config
	log      *logger.Logger
	closers  []context.CancelFunc
	reexec   reexecManager
}

type reexecManager interface {
	ReExec(argOverrides ...string)
}

// NewUpgrader creates an upgrader which is capable of performing upgrade operation
func NewUpgrader(settings *artifact.Config, log *logger.Logger, closers []context.CancelFunc, reexec reexecManager) *Upgrader {
	return &Upgrader{
		settings: settings,
		log:      log,
		closers:  closers,
		reexec:   reexec,
	}
}

// Upgrade upgrades running agent
func (u *Upgrader) Upgrade(ctx context.Context, version, sourceURI, actionID string) error {
	archivePath, err := u.downloadArtifact(ctx, version, sourceURI)
	if err != nil {
		return err
	}

	newHash, err := u.unpack(ctx, version, sourceURI, archivePath)
	if err != nil {
		return err
	}

	if newHash == "" {
		return errors.New("unknown hash")
	}

	if strings.HasPrefix(release.Commit(), newHash) {
		return errors.New("upgrading to same version")
	}

	if err := u.changeSymlink(ctx, newHash); err != nil {
		rollbackInstall(newHash)
		return err
	}

	if err := u.markUpgrade(ctx, version, newHash, actionID); err != nil {
		rollbackInstall(newHash)
		return err
	}

	u.reexec.ReExec()
	return nil
}

func isSubdir(base, target string) (bool, error) {
	relPath, err := filepath.Rel(base, target)
	return strings.HasPrefix(relPath, ".."), err
}

func rollbackInstall(hash string) {
	os.RemoveAll(filepath.Join(paths.Data(), fmt.Sprintf("%s-%s", agentName, hash)))
}
