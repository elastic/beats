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

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/install"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

const (
	agentName       = "elastic-agent"
	hashLen         = 6
	agentCommitFile = ".elastic-agent.active.commit"
)

var (
	agentSpec = program.Spec{
		Name:     "Elastic Agent",
		Cmd:      agentName,
		Artifact: "beats/" + agentName,
	}
)

// Upgrader performs an upgrade
type Upgrader struct {
	agentInfo   *info.AgentInfo
	settings    *artifact.Config
	log         *logger.Logger
	closers     []context.CancelFunc
	reexec      reexecManager
	acker       acker
	reporter    stateReporter
	upgradeable bool
}

// Action is the upgrade action state.
type Action interface {
	// Version to upgrade to.
	Version() string
	// SourceURI for download.
	SourceURI() string
	// FleetAction is the action from fleet that started the action (optional).
	FleetAction() *fleetapi.ActionUpgrade
}

type reexecManager interface {
	ReExec(argOverrides ...string)
}

type acker interface {
	Ack(ctx context.Context, action fleetapi.Action) error
	Commit(ctx context.Context) error
}

type stateReporter interface {
	OnStateChange(id string, name string, s state.State)
}

// NewUpgrader creates an upgrader which is capable of performing upgrade operation
func NewUpgrader(agentInfo *info.AgentInfo, settings *artifact.Config, log *logger.Logger, closers []context.CancelFunc, reexec reexecManager, a acker, r stateReporter) *Upgrader {
	return &Upgrader{
		agentInfo:   agentInfo,
		settings:    settings,
		log:         log,
		closers:     closers,
		reexec:      reexec,
		acker:       a,
		reporter:    r,
		upgradeable: getUpgradeable(),
	}
}

// Upgradeable returns true if the Elastic Agent can be upgraded.
func (u *Upgrader) Upgradeable() bool {
	return u.upgradeable
}

// Upgrade upgrades running agent
func (u *Upgrader) Upgrade(ctx context.Context, a Action, reexecNow bool) (err error) {
	// report failed
	defer func() {
		if err != nil {
			if action := a.FleetAction(); action != nil {
				u.reportFailure(ctx, action, err)
			}
		}
	}()

	if !u.upgradeable {
		return fmt.Errorf(
			"cannot be upgraded; must be installed with install sub-command and " +
				"running under control of the systems supervisor")
	}

	u.reportUpdating(a.Version())

	sourceURI, err := u.sourceURI(a.Version(), a.SourceURI())
	archivePath, err := u.downloadArtifact(ctx, a.Version(), sourceURI)
	if err != nil {
		return err
	}

	newHash, err := u.unpack(ctx, a.Version(), archivePath)
	if err != nil {
		return err
	}

	if newHash == "" {
		return errors.New("unknown hash")
	}

	if strings.HasPrefix(release.Commit(), newHash) {
		// not an error
		if action := a.FleetAction(); action != nil {
			u.ackAction(ctx, action)
		}
		u.log.Warn("upgrading to same version")
		return nil
	}

	if err := copyActionStore(newHash); err != nil {
		return errors.New(err, "failed to copy action store")
	}

	if err := u.changeSymlink(ctx, newHash); err != nil {
		rollbackInstall(newHash)
		return err
	}

	if err := u.markUpgrade(ctx, newHash, a); err != nil {
		rollbackInstall(newHash)
		return err
	}

	if reexecNow {
		u.reexec.ReExec()
	}
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

	if err := u.ackAction(ctx, marker.Action); err != nil {
		return err
	}

	marker.Acked = true
	markerBytes, err = yaml.Marshal(marker)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(markerFile, markerBytes, 0600)
}

func (u *Upgrader) sourceURI(version, retrievedURI string) (string, error) {
	if retrievedURI != "" {
		return retrievedURI, nil
	}

	return u.settings.SourceURI, nil
}

// ackAction is used for successful updates, it was either updated successfully or to the same version
// so we need to remove updating state and get prevent from receiving same update action again.
func (u *Upgrader) ackAction(ctx context.Context, action fleetapi.Action) error {
	if err := u.acker.Ack(ctx, action); err != nil {
		return err
	}

	if err := u.acker.Commit(ctx); err != nil {
		return err
	}

	u.reporter.OnStateChange(
		"",
		agentName,
		state.State{Status: state.Running},
	)

	return nil
}

// report failure is used when update process fails. action is acked so it won't be received again
// and state is changed to FAILED
func (u *Upgrader) reportFailure(ctx context.Context, action fleetapi.Action, err error) {
	// ack action
	u.acker.Ack(ctx, action)

	// report failure
	u.reporter.OnStateChange(
		"",
		agentName,
		state.State{Status: state.Failed, Message: err.Error()},
	)
}

// reportUpdating sets state of agent to updating.
func (u *Upgrader) reportUpdating(version string) {
	// report failure
	u.reporter.OnStateChange(
		"",
		agentName,
		state.State{Status: state.Updating, Message: fmt.Sprintf("Update to version '%s' started", version)},
	)
}

func rollbackInstall(hash string) {
	os.RemoveAll(filepath.Join(paths.Data(), fmt.Sprintf("%s-%s", agentName, hash)))
}

func getUpgradeable() bool {
	// only upgradeable if running from Agent installer and running under the
	// control of the system supervisor (or built specifically with upgrading enabled)
	return release.Upgradeable() || (install.RunningInstalled() && install.RunningUnderSupervisor())
}

func copyActionStore(newHash string) error {
	currentActionStorePath := info.AgentActionStoreFile()

	newHome := filepath.Join(filepath.Dir(paths.Home()), fmt.Sprintf("%s-%s", agentName, newHash))
	newActionStorePath := filepath.Join(newHome, filepath.Base(currentActionStorePath))

	currentActionStore, err := ioutil.ReadFile(currentActionStorePath)
	if os.IsNotExist(err) {
		// nothing to copy
		return nil
	}
	if err != nil {
		return err
	}

	return ioutil.WriteFile(newActionStorePath, currentActionStore, 0600)
}
