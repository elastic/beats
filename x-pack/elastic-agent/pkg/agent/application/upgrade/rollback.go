// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package upgrade

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/client"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/hashicorp/go-multierror"
)

const (
	watcherSubcommand  = "watch"
	maxRestartCount    = 5
	restartBackoffInit = 5 * time.Second
	restartBackoffMax  = 90 * time.Second
)

// Rollback rollbacks to previous version which was functioning before upgrade.
func Rollback(ctx context.Context, prevHash, currentHash string) error {
	// change symlink
	if err := ChangeSymlink(ctx, prevHash); err != nil {
		return err
	}

	// revert active commit
	if err := UpdateActiveCommit(prevHash); err != nil {
		return err
	}

	// Restart
	if err := restartAgent(ctx); err != nil {
		return err
	}

	// cleanup everything except version we're rolling back into
	return Cleanup(prevHash)
}

// Cleanup removes all artifacts and files related to a specified version.
func Cleanup(currentHash string) error {
	// remove upgrade marker
	if err := CleanMarker(); err != nil {
		return err
	}

	// remove data/elastic-agent-{hash}
	dataDir, err := os.Open(paths.Data())
	if err != nil {
		return err
	}

	subdirs, err := dataDir.Readdirnames(0)
	if err != nil {
		return err
	}

	dirPrefix := fmt.Sprintf("%s-", agentName)
	currentDir := fmt.Sprintf("%s-%s", agentName, currentHash)
	for _, dir := range subdirs {
		if dir == currentDir {
			continue
		}

		if !strings.HasPrefix(dir, dirPrefix) {
			continue
		}

		hashedDir := filepath.Join(paths.Data(), dir)
		if cleanupErr := os.RemoveAll(hashedDir); cleanupErr != nil {
			err = multierror.Append(err, cleanupErr)
		}
	}

	return err
}

// InvokeWatcher invokes an agent instance using watcher argument for watching behavior of
// agent during upgrade period.
func InvokeWatcher(log *logger.Logger) error {
	if !IsUpgradeable() {
		log.Debug("agent is not upgradable, not starting watcher")
		return nil
	}

	homeExePath := filepath.Join(paths.Home(), agentName)

	cmd := exec.Command(homeExePath, watcherSubcommand,
		"--path.config", paths.Config(),
		"--path.home", paths.Top(),
	)
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	log.Debugf("Starting watcher %v", cmd)
	return cmd.Start()
	// go func() {
	// 	<-time.After(15 * time.Second)
	// 	if cmd.Process != nil {
	// 		cmd.Process.Kill()
	// 	}
	// }()
	// o, err := cmd.CombinedOutput()
	// log.Error(">>> ", string(o))
	// log.Error(">>> ", err)

	// return cmd.Start()
}

func restartAgent(ctx context.Context) error {
	restartFn := func(ctx context.Context) error {
		c := client.New()
		err := c.Connect(ctx)
		if err != nil {
			return errors.New(err, "Failed communicating to running daemon", errors.TypeNetwork, errors.M("socket", control.Address()))
		}
		defer c.Disconnect()

		err = c.Restart(ctx)
		if err != nil {
			return errors.New(err, "Failed trigger restart of daemon")
		}

		return nil
	}

	signal := make(chan struct{})
	backExp := backoff.NewExpBackoff(signal, restartBackoffInit, restartBackoffMax)

	for i := maxRestartCount; i >= 1; i-- {
		backExp.Wait()
		err := restartFn(ctx)
		if err == nil {
			break
		}

		if i == 1 {
			return err
		}
	}

	close(signal)
	return nil
}
