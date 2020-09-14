// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package upgrade

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
)

// changeSymlink changes root symlink so it points to updated version
func (u *Upgrader) changeSymlink(ctx context.Context, newHash string) error {
	// create symlink to elastic-agent-{hash}
	hashedDir := fmt.Sprintf("%s-%s", agentName, newHash)
	originalPath := filepath.Join(paths.Home(), agentName)
	newPath := filepath.Join(paths.Data(), hashedDir, agentName)

	agentBakName := agentName + ".bak"
	bakNewPath := filepath.Join(paths.Data(), hashedDir, agentBakName)

	if err := os.Symlink(bakNewPath, originalPath); err != nil {
		return err
	}

	// safely rotate
	return file.SafeFileRotate(newPath, bakNewPath)
}
