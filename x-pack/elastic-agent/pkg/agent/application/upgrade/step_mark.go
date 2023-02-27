// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package upgrade

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

const markerFilename = ".update-marker"

// UpdateMarker is a marker holding necessary information about ongoing upgrade.
type UpdateMarker struct {
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
func (u *Upgrader) markUpgrade(ctx context.Context, hash string, action Action) error {
	prevVersion := release.Version()
	prevHash := release.Commit()
	if len(prevHash) > hashLen {
		prevHash = prevHash[:hashLen]
	}

	marker := UpdateMarker{
		Hash:        hash,
		UpdatedOn:   time.Now(),
		PrevVersion: prevVersion,
		PrevHash:    prevHash,
		Action:      action.FleetAction(),
	}

	markerBytes, err := yaml.Marshal(marker)
	if err != nil {
		return errors.New(err, errors.TypeConfig, "failed to parse marker file")
	}

	markerPath := markerFilePath()
	if err := ioutil.WriteFile(markerPath, markerBytes, 0600); err != nil {
		return errors.New(err, errors.TypeFilesystem, "failed to create update marker file", errors.M(errors.MetaKeyPath, markerPath))
	}

	if err := UpdateActiveCommit(hash); err != nil {
		return err
	}

	return nil
}

// UpdateActiveCommit updates active.commit file to point to active version.
func UpdateActiveCommit(hash string) error {
	activeCommitPath := filepath.Join(paths.Top(), agentCommitFile)
	if err := ioutil.WriteFile(activeCommitPath, []byte(hash), 0644); err != nil {
		return errors.New(err, errors.TypeFilesystem, "failed to update active commit", errors.M(errors.MetaKeyPath, activeCommitPath))
	}

	return nil
}

// CleanMarker removes a marker from disk.
func CleanMarker() error {
	markerFile := markerFilePath()
	if err := os.Remove(markerFile); !os.IsNotExist(err) {
		return err
	}

	return nil
}

// LoadMarker loads the update marker. If the file does not exist it returns nil
// and no error.
func LoadMarker() (*UpdateMarker, error) {
	markerFile := markerFilePath()
	markerBytes, err := ioutil.ReadFile(markerFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	marker := &UpdateMarker{}
	if err := yaml.Unmarshal(markerBytes, &marker); err != nil {
		return nil, err
	}

	return marker, nil
}

func saveMarker(marker *UpdateMarker) error {
	markerBytes, err := yaml.Marshal(marker)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(markerFilePath(), markerBytes, 0600)
}

func markerFilePath() string {
	return filepath.Join(paths.Data(), markerFilename)
}
