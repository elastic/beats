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

	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
)

// ChangeSymlink updates symlink paths to match current version.
func ChangeSymlink(ctx context.Context, targetHash string) error {
	// create symlink to elastic-agent-{hash}
	hashedDir := fmt.Sprintf("%s-%s", agentName, targetHash)

	symlinkPath := filepath.Join(paths.Top(), agentName)
	newPath := filepath.Join(paths.Top(), "data", hashedDir, agentName)

	// handle windows suffixes
	if runtime.GOOS == "windows" {
		symlinkPath += ".exe"
		newPath += ".exe"
	}

	prevNewPath := prevSymlinkPath()

	// remove symlink to avoid upgrade failures
	if err := os.Remove(prevNewPath); !os.IsNotExist(err) {
		return err
	}

	if err := os.Symlink(newPath, prevNewPath); err != nil {
		return errors.New(err, errors.TypeFilesystem, "failed to update agent symlink")
	}

	// safely rotate
	return file.SafeFileRotate(symlinkPath, prevNewPath)
}

func prevSymlinkPath() string {
	agentPrevName := agentName + ".prev"

	// handle windows suffixes
	if runtime.GOOS == "windows" {
		agentPrevName = agentName + ".exe.prev"
	}

	return filepath.Join(paths.Top(), agentPrevName)
}
