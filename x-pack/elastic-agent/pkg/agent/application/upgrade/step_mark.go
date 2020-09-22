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
	"runtime"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

const markerFilename = ".update-marker"

type updateMarker struct {
	// Hash agent is updated to
	Hash string `json:"hash" yaml:"hash"`
	//UpdatedOn marks a date when update happened
	UpdatedOn time.Time `json:"updated_on" yaml:"updated_on"`

	// PrevVersion is a version agent is updated from
	PrevVersion string `json:"prev_version" yaml:"prev_version"`
	// PrevHash is a hash agent is updated from
	PrevHash string `json:"prev_hash" yaml:"prev_hash"`

	// Acked is a flag marking whether or not action was acked
	Acked  bool                    `json:"acked" yaml:"acked"`
	Action *fleetapi.ActionUpgrade `json:"action" yaml:"action"`
}

// markUpgrade marks update happened so we can handle grace period
func (h *Upgrader) markUpgrade(ctx context.Context, hash string, action *fleetapi.ActionUpgrade) error {
	if err := updateHomePath(hash); err != nil {
		return err
	}

	prevVersion := release.Version()
	prevHash := release.Commit()
	if len(prevHash) > hashLen {
		prevHash = prevHash[:hashLen]
	}

	marker := updateMarker{
		Hash:        hash,
		UpdatedOn:   time.Now(),
		PrevVersion: prevVersion,
		PrevHash:    prevHash,
		Action:      action,
	}

	markerBytes, err := yaml.Marshal(marker)
	if err != nil {
		return errors.New(err, errors.TypeConfig, "failed to parse marker file")
	}

	markerPath := filepath.Join(paths.Data(), markerFilename)
	if err := ioutil.WriteFile(markerPath, markerBytes, 0600); err != nil {
		return errors.New(err, errors.TypeFilesystem, "failed to create update marker file", errors.M(errors.MetaKeyPath, markerPath))
	}

	activeCommitPath := filepath.Join(paths.Config(), agentCommitFile)
	if err := ioutil.WriteFile(activeCommitPath, []byte(hash), 0644); err != nil {
		return errors.New(err, errors.TypeFilesystem, "failed to update active commit", errors.M(errors.MetaKeyPath, activeCommitPath))
	}

	return nil
}

func updateHomePath(hash string) error {
	if err := createPathsSymlink(hash); err != nil {
		return errors.New(err, errors.TypeFilesystem, "failed to create paths symlink")
	}

	pathsMap := make(map[string]string)
	pathsFilepath := filepath.Join(paths.Data(), "paths.yml")

	pathsBytes, err := ioutil.ReadFile(pathsFilepath)
	if err != nil {
		return errors.New(err, errors.TypeConfig, "failed to read paths file")
	}

	if err := yaml.Unmarshal(pathsBytes, &pathsMap); err != nil {
		return errors.New(err, errors.TypeConfig, "failed to parse paths file")
	}

	pathsMap["path.home"] = filepath.Join(filepath.Dir(paths.Home()), fmt.Sprintf("%s-%s", agentName, hash))

	pathsBytes, err = yaml.Marshal(pathsMap)
	if err != nil {
		return errors.New(err, errors.TypeConfig, "failed to marshal paths file")
	}

	return ioutil.WriteFile(pathsFilepath, pathsBytes, 0740)
}

func createPathsSymlink(hash string) error {
	// only on windows, as windows resolves PWD using symlinks in a different way.
	// we create symlink for each versioned agent inside `data/` directory
	// on other systems path is shared
	if runtime.GOOS != "windows" {
		return nil
	}

	dir := filepath.Join(paths.Data(), fmt.Sprintf("%s-%s", agentName, hash))
	versionedPath := filepath.Join(dir, "data", "paths.yml")
	if err := os.MkdirAll(filepath.Dir(versionedPath), 0700); err != nil {
		return err
	}

	pathsCfgPath := filepath.Join(paths.Data(), "paths.yml")
	return os.Symlink(pathsCfgPath, versionedPath)
}
