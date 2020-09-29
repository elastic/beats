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

package diskqueue

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
	"github.com/elastic/beats/v7/libbeat/paths"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// Settings contains the configuration fields to create a new disk queue
// or open an existing one.
type Settings struct {
	// The path on disk of the queue's containing directory, which will be
	// created if it doesn't exist. Within the directory, the queue's state
	// is stored in state.dat and each segment's data is stored in
	// {segmentIndex}.seg
	// If blank, the default directory is "diskqueue" within the beat's data
	// directory.
	Path string

	// MaxBufferSize is the maximum number of bytes that the queue should
	// ever occupy on disk. A value of 0 means the queue can grow until the
	// disk is full (this is not recommended on a primary system disk).
	MaxBufferSize uint64

	// MaxSegmentSize is the maximum number of bytes that should be written
	// to a single segment file before creating a new one.
	MaxSegmentSize uint64

	// How many events will be read from disk while waiting for a consumer
	// request.
	ReadAheadLimit int

	// How many events will be queued in memory waiting to be written to disk.
	// This setting should rarely matter in practice, but if data is coming
	// in faster than it can be written to disk for an extended period,
	// this limit can keep it from overflowing memory.
	WriteAheadLimit int

	// A listener that should be sent ACKs when an event is successfully
	// written to disk.
	WriteToDiskListener queue.ACKListener
}

// userConfig holds the parameters for a disk queue that are configurable
// by the end user in the beats yml file.
type userConfig struct {
	Path            string            `config:"path"`
	MaxSize         cfgtype.ByteSize  `config:"max_size" validate:"required"`
	SegmentSize     *cfgtype.ByteSize `config:"segment_size"`
	ReadAheadLimit  *int              `config:"read_ahead"`
	WriteAheadLimit *int              `config:"write_ahead"`
}

func (c *userConfig) Validate() error {
	// If the segment size is explicitly specified, the total queue size must
	// be at least twice as large.
	if c.SegmentSize != nil && c.MaxSize != 0 && c.MaxSize < *c.SegmentSize*2 {
		return errors.New(
			"Disk queue max_size must be at least twice as big as segment_size")
	}

	// We require a total queue size of at least 10MB, and a segment size of
	// at least 1MB. The queue can support lower thresholds, but it will perform
	// terribly, so we give an explicit error in that case.
	// These bounds are still extremely low for Beats ingestion, but if all you
	// need is for a low-volume stream on a tiny device to persist between
	// restarts, it will work fine.
	if c.MaxSize != 0 && c.MaxSize < 10*1000*1000 {
		return fmt.Errorf(
			"Disk queue max_size (%d) cannot be less than 10MB", c.MaxSize)
	}
	if c.SegmentSize != nil && *c.SegmentSize < 1000*1000 {
		return fmt.Errorf(
			"Disk queue segment_size (%d) cannot be less than 1MB", *c.SegmentSize)
	}

	return nil
}

// DefaultSettings returns a Settings object with reasonable default values
// for all important fields.
func DefaultSettings() Settings {
	return Settings{
		MaxSegmentSize: 100 * (1 << 20), // 100MiB
		MaxBufferSize:  (1 << 30),       // 1GiB

		ReadAheadLimit:  512,
		WriteAheadLimit: 2048,
	}
}

// SettingsForUserConfig returns a Settings struct initialized with the
// end-user-configurable settings in the given config tree.
func SettingsForUserConfig(config *common.Config) (Settings, error) {
	userConfig := userConfig{}
	if err := config.Unpack(&userConfig); err != nil {
		return Settings{}, fmt.Errorf("parsing user config: %w", err)
	}
	settings := DefaultSettings()
	settings.Path = userConfig.Path

	settings.MaxBufferSize = uint64(userConfig.MaxSize)
	if userConfig.SegmentSize != nil {
		settings.MaxSegmentSize = uint64(*userConfig.SegmentSize)
	} else {
		// If no value is specified, default segment size is total queue size
		// divided by 10.
		settings.MaxSegmentSize = uint64(userConfig.MaxSize) / 10
	}

	if userConfig.ReadAheadLimit != nil {
		settings.ReadAheadLimit = *userConfig.ReadAheadLimit
	}
	if userConfig.WriteAheadLimit != nil {
		settings.WriteAheadLimit = *userConfig.WriteAheadLimit
	}

	return settings, nil
}

//
// bookkeeping helpers
//

func (settings Settings) directoryPath() string {
	if settings.Path == "" {
		return paths.Resolve(paths.Data, "diskqueue")
	}
	return settings.Path
}

func (settings Settings) stateFilePath() string {
	return filepath.Join(settings.directoryPath(), "state.dat")
}

func (settings Settings) segmentPath(segmentID segmentID) string {
	return filepath.Join(
		settings.directoryPath(),
		fmt.Sprintf("%v.seg", segmentID))
}

func (settings Settings) maxSegmentOffset() segmentOffset {
	return segmentOffset(settings.MaxSegmentSize - segmentHeaderSize)
}
