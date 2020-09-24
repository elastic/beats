// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package upgrade

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
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
	acker    acker
}

type reexecManager interface {
	ReExec(argOverrides ...string)
}

type acker interface {
	Ack(ctx context.Context, action fleetapi.Action) error
	Commit(ctx context.Context) error
}

// NewUpgrader creates an upgrader which is capable of performing upgrade operation
func NewUpgrader(settings *artifact.Config, log *logger.Logger, closers []context.CancelFunc, reexec reexecManager, a acker) *Upgrader {
	return &Upgrader{
		settings: settings,
		log:      log,
		closers:  closers,
		reexec:   reexec,
		acker:    a,
	}
}

// Upgrade upgrades running agent
func (u *Upgrader) Upgrade(ctx context.Context, a *fleetapi.ActionUpgrade) error {
	archivePath, err := u.downloadArtifact(ctx, a.Version, a.SourceURI)
	if err != nil {
		return err
	}

	newHash, err := u.unpack(ctx, a.Version, a.SourceURI, archivePath)
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

	if err := u.markUpgrade(ctx, newHash, a); err != nil {
		rollbackInstall(newHash)
		return err
	}

	u.reexec.ReExec()
	return nil
}

// Ack acks last upgrade action
func (u *Upgrader) Ack(ctx context.Context) error {
	// get upgrade action
	markerFile := filepath.Join(paths.Data(), markerFilename)
	markerBytes, err := ioutil.ReadFile(markerFile)
	if err != nil && os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	marker := &updateMarker{}
	if err := yaml.Unmarshal(markerBytes, marker); err != nil {
		return err
	}

	if marker.Acked {
		return nil
	}

	if err := u.acker.Ack(ctx, marker.Action); err != nil {
		return err
	}

	if err := u.acker.Commit(ctx); err != nil {
		return err
	}

	marker.Acked = true
	markerBytes, err = yaml.Marshal(marker)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(markerFile, markerBytes, 0600)
}

func isSubdir(base, target string) (bool, error) {
	relPath, err := filepath.Rel(base, target)
	return strings.HasPrefix(relPath, ".."), err
}

func rollbackInstall(hash string) {
	os.RemoveAll(filepath.Join(paths.Data(), fmt.Sprintf("%s-%s", agentName, hash)))
}
