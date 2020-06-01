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
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
)

// This is the queue metadata that is saved to disk. Currently it only
// tracks the read position in the queue; all other data is contained
// in the segment files.
type diskQueuePersistentState struct {
	// The schema version for the state file (currently always 0).
	version uint32

	// The oldest position in the queue. This is advanced as we receive ACKs from
	// downstream consumers indicating it is safe to remove old events.
	firstPosition bufferPosition
}

// A wrapper around os.File that caches the most recently read / written
// state data.
type stateFile struct {
	// An open file handle to the queue's state file.
	file *os.File

	// A pointer to the disk queue state that was read when this queue was
	// opened, or nil if a new state file was created.
	loadedState *diskQueuePersistentState

	// If there was a non-fatal error loading the queue state, it is stored
	// here. In this case, the queue overwrites the existing state file with
	// a valid starting state.
	stateErr error
}

// Given an open file handle, decode the file as a diskQueuePersistentState
// and return the result if successful, otherwise an error.
func persistentStateFromHandle(
	file *os.File,
) (*diskQueuePersistentState, error) {
	_, err := file.Seek(0, 0)
	if err != nil {
		return nil, err
	}
	state := diskQueuePersistentState{}

	reader := bufio.NewReader(file)
	err = binary.Read(reader, binary.LittleEndian,
		&state.version)
	if err != nil {
		return nil, err
	}

	err = binary.Read(reader, binary.LittleEndian,
		&state.firstPosition.segmentIndex)
	if err != nil {
		return nil, err
	}

	err = binary.Read(reader, binary.LittleEndian,
		&state.firstPosition.byteIndex)
	if err != nil {
		return nil, err
	}

	return &state, nil
}

// Given an open file handle and the first remaining position of a disk queue,
// binary encode the corresponding diskQueuePersistentState and overwrite the
// file with the result. Returns nil if successful, otherwise an error.
func writePersistentStateToHandle(
	file *os.File,
	firstPosition bufferPosition,
) error {
	_, err := file.Seek(0, 0)
	if err != nil {
		return err
	}

	var version uint32 = 0
	err = binary.Write(file, binary.LittleEndian,
		&version)
	if err != nil {
		return err
	}

	err = binary.Write(file, binary.LittleEndian,
		&firstPosition.segmentIndex)
	if err != nil {
		return err
	}

	err = binary.Write(file, binary.LittleEndian,
		&firstPosition.byteIndex)
	if err != nil {
		return err
	}

	return nil
}

func (stateFile *stateFile) Close() error {
	return stateFile.file.Close()
}

func stateFileForPath(path string) (*stateFile, error) {
	var state *diskQueuePersistentState
	var stateErr error
	// Try to open an existing state file.
	file, err := os.OpenFile(path, os.O_RDWR, 0600)
	if err != nil {
		// If we can't open the file, it's likely a new queue, so try to create it.
		file, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			return nil, fmt.Errorf("Couldn't open disk queue metadata file: %w", err)
		}
	} else {
		// Read the existing state.
		state, stateErr = persistentStateFromHandle(file)
		// This
		if err != nil {
			// TODO: this shouldn't be a fatal error. If the state file exists but
			// its contents are invalid, then we should log a warning and overwrite
			// it with metadata derived from the segment files instead.
			return nil, err
		}
	}
	result := &stateFile{
		file:        file,
		loadedState: state,
		stateErr:    stateErr,
	}
	if state == nil {
		// Initialize with new zero state.
		err = writePersistentStateToHandle(file, bufferPosition{0, 0})
		if err != nil {
			return nil, fmt.Errorf("Couldn't write queue state to disk: %w", err)
		}
	}
	return result, nil
}
