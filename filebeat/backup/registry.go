// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package backup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	regLogFilename = "log.json"
)

var (
	errCheckpointNotFound = errors.New("there is no checkpoint in the registry")
	checkpointFileRegexp  = regexp.MustCompile(`^[0-9]+\.json$`)
)

// NewRegistryBackuper creates a new backuper that creates backups for the registry files.
// It creates backups for the checkpoint file (if exists) and the registry log.
// `regHome` must be the final directory with the registry log (`log.json`).
func NewRegistryBackuper(log *logp.Logger, regHome string) Backuper {
	return &registryBackuper{
		log:     log,
		regHome: regHome,
	}
}

type registryBackuper struct {
	log             *logp.Logger
	regHome         string
	removeCallbacks []func() error
}

// Backup backs up the active checkpoint if any and the registry log file
func (rb *registryBackuper) Backup() error {
	var toBackup []string

	rb.log.Info("Attempting to find the checkpoint...")
	checkpoint, err := rb.findCheckpoint()
	if err == nil {
		toBackup = append(toBackup, checkpoint)
		rb.log.Infof("Found checkpoint at %s", checkpoint)
	} else if err != nil && !errors.Is(err, errCheckpointNotFound) {
		return fmt.Errorf("failed to look for a checkpoint file in %s: %w", rb.regHome, err)
	} else {
		rb.log.Info("Checkpoint not found")
	}

	registryLog := filepath.Join(rb.regHome, regLogFilename)
	rb.log.Infof("Checking if the registry log exists at %s...", registryLog)
	exists, err := fileExists(registryLog)
	if err != nil {
		return fmt.Errorf("failed to look for a registry log file %s: %w", registryLog, err)
	}
	if exists {
		toBackup = append(toBackup, registryLog)
		rb.log.Infof("Found the registry log at %s", registryLog)
	}

	rb.log.Infof("Creating backups for %v...", toBackup)

	fb := NewFileBackuper(rb.log, toBackup)
	rb.removeCallbacks = append(rb.removeCallbacks, fb.Remove)

	return fb.Backup()
}

// Remove removes all registry backup files created by this backuper
func (rb registryBackuper) Remove() error {
	rb.log.Infof("Removing %d created backups...", len(rb.removeCallbacks))

	var errs []error
	for _, cb := range rb.removeCallbacks {
		err := cb()
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) != 0 {
		return fmt.Errorf("failed to remove registry backup: %v", errs)
	}

	return nil
}

// findCheckpoint finds the active checkpoint file if any.
// Returns `errCheckpointNotFound` if no checkpoint to be found
func (rb registryBackuper) findCheckpoint() (string, error) {
	entries, err := os.ReadDir(rb.regHome)
	if err != nil {
		return "", fmt.Errorf("failed to read the directory %s: %w", rb.regHome, err)
	}
	var checkpoints []string
	for _, entry := range entries {
		name := entry.Name()
		if !checkpointFileRegexp.MatchString(name) {
			continue
		}
		checkpoints = append(checkpoints, name)
	}

	if len(checkpoints) == 0 {
		return "", errCheckpointNotFound
	}

	// the latest checkpoint should be on the top, which means
	// the greatest filename wins (the latest operation number)
	sort.Slice(checkpoints, func(i, j int) bool {
		// `checkpointFileRegexp` guarantees the base file name is a number
		c1, _ := strconv.ParseInt(strings.TrimSuffix(filepath.Base(checkpoints[i]), ".json"), 10, 64)
		c2, _ := strconv.ParseInt(strings.TrimSuffix(filepath.Base(checkpoints[j]), ".json"), 10, 64)
		return c1 > c2
	})

	checkpoint := checkpoints[0]
	return filepath.Join(rb.regHome, checkpoint), nil
}
