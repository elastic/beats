// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package rollback

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/upgrade"
)

const (
	agentName         = "elastic-agent"
	watcherSubcommand = "watch"
)

// Rollback rollbacks to previous version which was functioning before upgrade.
func Rollback(ctx context.Context, prevHash, currentHash string) error {
	// action_store ??

	// change symlink
	if err := upgrade.ChangeSymlink(ctx, prevHash); err != nil {
		return err
	}

	// revert active commit
	if err := upgrade.UpdateActiveCommit(prevHash); err != nil {
		return err
	}

	return Cleanup(currentHash)
}

// Cleanup removes all artifacts and files related to a specified version.
func Cleanup(prevHash string) error {
	// remove upgrade marker
	if err := upgrade.CleanMarker(); err != nil {
		return err
	}

	// remove data/elastic-agent-{hash}
	hashedDir := filepath.Join(paths.Data(), fmt.Sprintf("%s-%s", agentName, prevHash))
	if err := os.RemoveAll(hashedDir); err != nil {
		return err
	}

	return nil
}

// InvokeWatcher invokes an agent instance using watcher argument for watching behavior of
// agent during upgrade period.
func InvokeWatcher() error {
	if !upgrade.IsUpgradeable() {
		return nil
	}

	topExePath := filepath.Join(paths.Top(), agentName)

	cmd := exec.Command(topExePath, watcherSubcommand)
	cmd.Dir = paths.Top()

	return cmd.Start()
}
