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

package awss3

import (
	"fmt"
	"time"
)

// State is used to communicate the reading state of a file
type State struct {
	Bucket       string    `json:"bucket" struct:"bucket"`
	Key          string    `json:"key" struct:"key"`
	LastModified time.Time `json:"last_modified" struct:"last_modifed"`
	Size         int64     `json:"size" struct:"size"`
	Offset       int64     `json:"offset" struct:"offset"`
}

// NewState creates a new s3 object state
func NewState(bucket, key string, size int64, lastModified time.Time) State {
	s := State{
		Bucket:       bucket,
		Key:          key,
		LastModified: lastModified,
		Size:         size,
		Offset:       0,
	}

	return s
}

func (s *State) Update(offset int64) {
	s.Offset = offset
}

// IsEqual checks if the two states point to the same file.
func (s *State) IsEqual(c *State) bool {
	return s.Bucket == c.Bucket && s.Key == c.Key && s.Size == c.Size && s.LastModified.Equal(c.LastModified)
}

// String returns string representation of the struct
func (s *State) String() string {
	return fmt.Sprintf(
		"{Key: %v, Size: %v, Offset: %v, LastModified: %v}",
		s.Key,
		s.Size,
		s.Offset,
		s.LastModified)
}
