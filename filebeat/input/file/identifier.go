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
	"strconv"
	"strings"

	"github.com/mitchellh/hashstructure"

	conf "github.com/elastic/elastic-agent-libs/config"
)

const (
	nativeName      = "native"
	pathName        = "path"
	inodeMarkerName = "inode_marker"

	DefaultIdentifierName = nativeName
	identitySep           = "::"
)

var identifierFactories = map[string]IdentifierFactory{
	nativeName:      newINodeDeviceIdentifier,
	pathName:        newPathIdentifier,
	inodeMarkerName: newINodeMarkerIdentifier,
}

type IdentifierFactory func(*conf.C) (StateIdentifier, error)

// StateIdentifier generates an ID for a State.
type StateIdentifier interface {
	// GenerateID generates and returns the ID of the state and its type
	GenerateID(State) (id, identifierType string)
}

// NewStateIdentifier creates a new state identifier for a log input.
func NewStateIdentifier(ns *conf.Namespace) (StateIdentifier, error) {
	if ns == nil {
		return newINodeDeviceIdentifier(nil)
	}

	identifierType := ns.Name()
	f, ok := identifierFactories[identifierType]
	if !ok {
		return nil, fmt.Errorf("no such file_identity generator: %s", identifierType)
	}

	return f(ns.Config())
}

type inodeDeviceIdentifier struct {
	name string
}

func newINodeDeviceIdentifier(_ *conf.C) (StateIdentifier, error) {
	return &inodeDeviceIdentifier{
		name: nativeName,
	}, nil
}

func (i *inodeDeviceIdentifier) GenerateID(s State) (id, identifierType string) {
	stateID := i.name + identitySep + s.FileStateOS.String()
	return genIDWithHash(s.Meta, stateID), i.name
}

type pathIdentifier struct {
	name string
}

func newPathIdentifier(_ *conf.C) (StateIdentifier, error) {
	return &pathIdentifier{
		name: pathName,
	}, nil
}

func (p *pathIdentifier) GenerateID(s State) (id, identifierType string) {
	stateID := p.name + identitySep + s.Source
	return genIDWithHash(s.Meta, stateID), p.name
}

func genIDWithHash(meta map[string]string, fileID string) string {
	if len(meta) == 0 {
		return fileID
	}

	hashValue, _ := hashstructure.Hash(meta, nil)
	var hashBuf [17]byte
	hash := strconv.AppendUint(hashBuf[:0], hashValue, 16)
	hash = append(hash, '-')

	var b strings.Builder
	b.Grow(len(hash) + len(fileID))
	b.Write(hash)
	b.WriteString(fileID)

	return b.String()
}

// mockIdentifier is used for testing
type MockIdentifier struct{}

func (m *MockIdentifier) GenerateID(s State) (string, string) { return s.Id, "mock" }
