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

// Given an open file handle to the queue state, decode the current position
// and return the result if successful, otherwise an error.
func queuePositionFromHandle(
	file *os.File,
) (queuePosition, error) {
	_, err := file.Seek(0, 0)
	if err != nil {
		return queuePosition{}, err
	}

	reader := bufio.NewReader(file)
	var version uint32
	err = binary.Read(reader, binary.LittleEndian, &version)
	if err != nil {
		return queuePosition{}, err
	}
	if version != 0 {
		return queuePosition{},
			fmt.Errorf("Unsupported queue metadata version (%d)", version)
	}

	position := queuePosition{}
	err = binary.Read(reader, binary.LittleEndian, &position.segmentID)
	if err != nil {
		return queuePosition{}, err
	}

	err = binary.Read(
		reader, binary.LittleEndian, &position.offset)
	if err != nil {
		return queuePosition{}, err
	}

	return position, nil
}

func queuePositionFromPath(path string) (queuePosition, error) {
	// Try to open an existing state file.
	file, err := os.OpenFile(path, os.O_RDONLY, 0600)
	if err != nil {
		return queuePosition{}, err
	}
	defer file.Close()
	return queuePositionFromHandle(file)
}

// Given the queue position, encode and write it to the given file handle.
// Returns nil if successful, otherwise an error.
func writeQueuePositionToHandle(
	file *os.File,
	position queuePosition,
) error {
	_, err := file.Seek(0, 0)
	if err != nil {
		return err
	}

	// Want to write: version (0), segment id, segment offset.
	err = binary.Write(file, binary.LittleEndian, uint32(0))
	if err != nil {
		return err
	}
	err = binary.Write(file, binary.LittleEndian, position.segmentID)
	if err != nil {
		return err
	}
	err = binary.Write(file, binary.LittleEndian, position.offset)
	return err
}
