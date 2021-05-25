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

package filestream

import (
	"fmt"
	"os"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/file"
)

type identifierFeature uint8

const (
	// trackRename is a feature of an identifier which changes
	// IDs if a source is renamed.
	trackRename identifierFeature = iota

	nativeName      = "native"
	pathName        = "path"
	inodeMarkerName = "inode_marker"

	DefaultIdentifierName = nativeName
	identitySep           = "::"
)

var (
	identifierFactories = map[string]identifierFactory{
		nativeName:      newINodeDeviceIdentifier,
		pathName:        newPathIdentifier,
		inodeMarkerName: newINodeMarkerIdentifier,
	}
)

type identifierFactory func(*common.Config) (fileIdentifier, error)

type fileIdentifier interface {
	GetSource(loginp.FSEvent) fileSource
	Name() string
	Supports(identifierFeature) bool
}

// fileSource implements the Source interface
// It is required to identify and manage file sources.
type fileSource struct {
	info      os.FileInfo
	newPath   string
	oldPath   string
	truncated bool

	name                string
	identifierGenerator string
}

// Name returns the registry identifier of the file.
func (f fileSource) Name() string {
	return f.name
}

// newFileIdentifier creates a new state identifier for a log input.
func newFileIdentifier(ns *common.ConfigNamespace) (fileIdentifier, error) {
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

func newINodeDeviceIdentifier(_ *common.Config) (fileIdentifier, error) {
	return &inodeDeviceIdentifier{
		name: nativeName,
	}, nil
}

func (i *inodeDeviceIdentifier) GetSource(e loginp.FSEvent) fileSource {
	return fileSource{
		info:                e.Info,
		newPath:             e.NewPath,
		oldPath:             e.OldPath,
		truncated:           e.Op == loginp.OpTruncate,
		name:                i.name + identitySep + file.GetOSState(e.Info).String(),
		identifierGenerator: i.name,
	}
}

func (i *inodeDeviceIdentifier) Name() string {
	return i.name
}

func (i *inodeDeviceIdentifier) Supports(f identifierFeature) bool {
	switch f {
	case trackRename:
		return true
	default:
	}
	return false
}

type pathIdentifier struct {
	name string
}

func newPathIdentifier(_ *common.Config) (fileIdentifier, error) {
	return &pathIdentifier{
		name: pathName,
	}, nil
}

func (p *pathIdentifier) GetSource(e loginp.FSEvent) fileSource {
	path := e.NewPath
	if e.Op == loginp.OpDelete {
		path = e.OldPath
	}
	return fileSource{
		info:                e.Info,
		newPath:             e.NewPath,
		oldPath:             e.OldPath,
		truncated:           e.Op == loginp.OpTruncate,
		name:                p.name + identitySep + path,
		identifierGenerator: p.name,
	}
}

func (p *pathIdentifier) Name() string {
	return p.name
}

func (p *pathIdentifier) Supports(f identifierFeature) bool {
	return false
}

// mockIdentifier is used for testing
type MockIdentifier struct{}

func (m *MockIdentifier) GetSource(e loginp.FSEvent) fileSource {
	return fileSource{identifierGenerator: "mock"}
}

func (m *MockIdentifier) Name() string { return "mock" }

func (m *MockIdentifier) Supports(_ identifierFeature) bool { return false }
