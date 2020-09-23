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

// changeSymlink changes root symlink so it points to updated version
func (u *Upgrader) changeSymlink(ctx context.Context, newHash string) error {
	// create symlink to elastic-agent-{hash}
	hashedDir := fmt.Sprintf("%s-%s", agentName, newHash)

	agentBakName := agentName + ".prev"
	symlinkPath := filepath.Join(paths.Top(), agentName)
	newPath := filepath.Join(paths.Top(), "data", hashedDir, agentName)

	// handle windows suffixes
	if runtime.GOOS == "windows" {
		agentBakName = agentName + ".exe.prev" //.bak is already used
		symlinkPath += ".exe"
		newPath += ".exe"
	}

	bakNewPath := filepath.Join(paths.Top(), agentBakName)
	if err := os.Symlink(newPath, bakNewPath); err != nil {
		return errors.New(err, errors.TypeFilesystem, "failed to update agent symlink")
	}

	// safely rotate
	return file.SafeFileRotate(symlinkPath, bakNewPath)
}
