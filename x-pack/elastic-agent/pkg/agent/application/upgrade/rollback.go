// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package upgrade

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

	// TODO: resurrect action store?

	// Restart
	if err := restartAgent(ctx); err != nil {
		return err
	}

	// cleanup everything except version we're rolling back into
	return Cleanup(prevHash, true)
}

// Cleanup removes all artifacts and files related to a specified version.
func Cleanup(currentHash string, removeMarker bool) error {
	<-time.After(afterRestartDelay)

	// remove upgrade marker
	if removeMarker {
		if err := CleanMarker(); err != nil {
			return err
		}
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
		if cleanupErr := os.RemoveAll(hashedDir); cleanupErr != nil && !isErrorExpected(cleanupErr) {
			err = multierror.Append(err, cleanupErr)
		}
	}

	return err
}

func isErrorExpected(err error) bool {
	// cannot remove self, this is expected
	// fails with  remove {path}}\elastic-agent.exe: Access is denied
	if runtime.GOOS == "windows" && strings.Contains(err.Error(), "elastic-agent.exe") && strings.Contains(err.Error(), "Access is denied") {
		return true
	}
	return false
}

// InvokeWatcher invokes an agent instance using watcher argument for watching behavior of
// agent during upgrade period.
func InvokeWatcher(log *logger.Logger) error {
	if !IsUpgradeable() {
		log.Debug("agent is not upgradable, not starting watcher")
		return nil
	}

	cmd := invokeCmd()
	defer func() {
		if cmd.Process != nil {
			log.Debugf("releasing watcher %v", cmd.Process.Pid)
			cmd.Process.Release()
		}
	}()

	log.Debugf("Starting watcher %v", cmd)
	return cmd.Start()

	// TODO: remove me
	// var cred = &syscall.Credential{
	// 	Uid:         uint32(os.Getuid()),
	// 	Gid:         uint32(os.Getgid()),
	// 	Groups:      nil,
	// 	NoSetGroups: true,
	// }

	// var sysproc = &syscall.SysProcAttr{
	// 	Credential: cred,
	// 	Setsid:     true,
	// 	// Setpgid:    true,
	// }
	// var attr = os.ProcAttr{
	// 	Dir: paths.Top(),
	// 	Env: os.Environ(),
	// 	Files: []*os.File{
	// 		os.Stdin,
	// 		os.Stdout,
	// 		os.Stderr,
	// 	},
	// 	Sys: sysproc,
	// }

	// args := []string{watcherSubcommand,
	// 	"--path.config", paths.Config(),
	// 	"--path.home", paths.Top(),
	// }
	// log.Error("starting watcher")
	// _, err := os.StartProcess(homeExePath, args, &attr)
	// if err != nil {
	// 	log.Error("failed to invoke watcher", err)
	// 	return err
	// }

	// // if err := process.Release(); err != nil {
	// // 	log.Error("failed to release watcher", err)
	// // 	return err
	// // }

	// return nil
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
