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

package file

import (
	"fmt"
	"os"
	"time"

	"github.com/menderesk/beats/v7/libbeat/common/file"
)

// State is used to communicate the reading state of a file
type State struct {
	Id             string            `json:"id" struct:"id"`
	PrevId         string            `json:"prev_id" struct:"prev_id"`
	Finished       bool              `json:"-" struct:"-"` // harvester state
	Fileinfo       os.FileInfo       `json:"-" struct:"-"` // the file info
	Source         string            `json:"source" struct:"source"`
	Offset         int64             `json:"offset" struct:"offset"`
	Timestamp      time.Time         `json:"timestamp" struct:"timestamp"`
	TTL            time.Duration     `json:"ttl" struct:"ttl"`
	Type           string            `json:"type"  struct:"type"`
	Meta           map[string]string `json:"meta" struct:"meta,omitempty"`
	FileStateOS    file.StateOS      `json:"FileStateOS" struct:"FileStateOS"`
	IdentifierName string            `json:"identifier_name" struct:"identifier_name"`
}

// NewState creates a new file state
func NewState(fileInfo os.FileInfo, path string, t string, meta map[string]string, identifier StateIdentifier) State {
	if len(meta) == 0 {
		meta = nil
	}

	s := State{
		Fileinfo:    fileInfo,
		Source:      path,
		Finished:    false,
		FileStateOS: file.GetOSState(fileInfo),
		Timestamp:   time.Now(),
		TTL:         -1, // By default, state does have an infinite ttl
		Type:        t,
		Meta:        meta,
	}

	s.Id, s.IdentifierName = identifier.GenerateID(s)

	return s
}

// IsEqual checks if the two states point to the same file.
func (s *State) IsEqual(c *State) bool {
	return s.Id == c.Id
}

// String returns string representation of the struct
func (s *State) String() string {
	return fmt.Sprintf(
		"{Id: %v, Finished: %v, Fileinfo: %v, Source: %v, Offset: %v, Timestamp: %v, TTL: %v, Type: %v, Meta: %v, FileStateOS: %v}",
		s.Id,
		s.Finished,
		s.Fileinfo,
		s.Source,
		s.Offset,
		s.Timestamp,
		s.TTL,
		s.Type,
		s.Meta,
		s.FileStateOS)
}
