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
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const (
	inodeDeviceIDName = "inode_deviceid"
	pathName          = "path"
	inodeMarkerName   = "inode_marker"

	DefaultIdentifierName = inodeDeviceIDName
	identitySep           = "::"
)

var (
	identifierFactories = map[string]IdentifierFactory{
		inodeDeviceIDName: newINodeDeviceIdentifier,
		pathName:          newPathIdentifier,
		inodeMarkerName:   newINodeMarkerIdentifier,
	}
)

type IdentifierFactory func(*common.Config) (StateIdentifier, error)

// StateIdentifier generates an ID for a State.
type StateIdentifier interface {
	// GenerateID generates and returns the ID of the state and its type
	GenerateID(State) (id, identifierType string)
}

// NewStateIdentifier creates a new state identifier for a log input.
func NewStateIdentifier(ns *common.ConfigNamespace) (StateIdentifier, error) {
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

func newINodeDeviceIdentifier(_ *common.Config) (StateIdentifier, error) {
	return &inodeDeviceIdentifier{
		name: inodeDeviceIDName,
	}, nil
}

func (i *inodeDeviceIdentifier) GenerateID(s State) (id, identifierType string) {
	stateID := i.name + identitySep + s.FileStateOS.String()
	return genIDWithHash(s.Meta, stateID), i.name
}

type pathIdentifier struct {
	name string
}

func newPathIdentifier(_ *common.Config) (StateIdentifier, error) {
	return &pathIdentifier{
		name: pathName,
	}, nil
}

func (p *pathIdentifier) GenerateID(s State) (id, identifierType string) {
	stateID := p.name + identitySep + s.Source
	return genIDWithHash(s.Meta, stateID), p.name
}

type inodeMarkerIdentifier struct {
	log        *logp.Logger
	name       string
	markerPath string

	markerFileLastModifitaion time.Time
	markerTxt                 string
}

func newINodeMarkerIdentifier(cfg *common.Config) (StateIdentifier, error) {
	var config struct {
		MarkerPath string `config:"path" validate:"required"`
	}
	err := cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("error while reading configuration of INode + marker file configuration: %v", err)
	}

	fi, err := os.Stat(config.MarkerPath)
	if err != nil {
		return nil, fmt.Errorf("error while opening marker file at %s: %v", config.MarkerPath, err)
	}
	markerContent, err := ioutil.ReadFile(config.MarkerPath)
	if err != nil {
		return nil, fmt.Errorf("error while reading marker file at %s: %v", config.MarkerPath, err)
	}
	return &inodeMarkerIdentifier{
		log:                       logp.NewLogger("inode_marker_identifier_" + filepath.Base(config.MarkerPath)),
		name:                      inodeMarkerName,
		markerPath:                config.MarkerPath,
		markerFileLastModifitaion: fi.ModTime(),
		markerTxt:                 string(markerContent),
	}, nil
}

func (i *inodeMarkerIdentifier) markerContents() string {
	f, err := os.Open(i.markerPath)
	if err != nil {
		i.log.Errorf("Failed to open marker file %s: %v", i.markerPath, err)
		return ""
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		i.log.Errorf("Failed to fetch file information for %s: %v", i.markerPath, err)
		return ""
	}
	if i.markerFileLastModifitaion.Before(fi.ModTime()) {
		contents, err := ioutil.ReadFile(i.markerPath)
		if err != nil {
			i.log.Errorf("Error while reading contents of marker file: %v", err)
			return ""
		}
		i.markerTxt = string(contents)
	}

	return i.markerTxt
}

func (i *inodeMarkerIdentifier) GenerateID(s State) (id, identifierType string) {
	m := i.markerContents()

	stateID := fmt.Sprintf("%s%s%s-%s", i.name, identitySep, s.FileStateOS.InodeString(), m)
	return genIDWithHash(s.Meta, stateID), i.name
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
