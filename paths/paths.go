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

// Package paths provides a common way to handle paths
// configuration for all Beats.
//
// Currently the following paths are defined:
//
// path.home - It’s the default folder for everything that doesn't fit in
// the categories below
//
// path.data - Contains things that are expected to change often during normal
// operations (“registry” files, UUID file, etc.)
//
// path.config - Configuration files and Elasticsearch template default location
//
// The owning application decides how these values are populated, for example
// from CLI flags or its configuration file.
//
// Use the Resolve method on a *Path to resolve files to their absolute paths.
// For example, to look for a file in the config path:
//
// cfgfilePath := p.Resolve(paths.Config, "beat.yml")
package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// Path tracks user-configurable path locations and directories
type Path struct {
	Home   string
	Config string
	Data   string
	Logs   string
}

// FileType is an enumeration type representing the file types.
// Currently existing file types are: Home, Config, Data
type FileType string

const (
	// Home is the "root" directory for the running beats instance
	Home FileType = "home"
	// Config is the path to the beat config
	Config FileType = "config"
	// Data is the path to the beat data directory
	Data FileType = "data"
	// Logs is the path to the beats logs directory
	Logs FileType = "logs"
)

// New returns a Path with all values empty.
func New() *Path {
	return &Path{}
}

// InitPaths copies cfg into paths, filling unset values with defaults
// derived from Home. It also creates the data directory on disk and
// returns an error on failure.
func (paths *Path) InitPaths(cfg *Path) error {
	err := paths.initPaths(cfg)
	if err != nil {
		return err
	}

	// make sure the data path exists
	err = os.MkdirAll(paths.Data, 0770)
	if err != nil {
		return fmt.Errorf("failed to create data path %s: %w", paths.Data, err)
	}

	return nil
}

// initPaths copies cfg into paths and fills unset values with defaults derived from Home.
func (paths *Path) initPaths(cfg *Path) error {
	*paths = *cfg

	// default for config path
	if paths.Config == "" {
		paths.Config = paths.Home
	}

	// default for data path
	if paths.Data == "" {
		paths.Data = filepath.Join(paths.Home, "data")
	}

	// default for logs path
	if paths.Logs == "" {
		paths.Logs = filepath.Join(paths.Home, "logs")
	}

	return nil
}

// Resolve resolves a path to a location in one of the default
// folders. For example, Resolve(Home, "test") returns an absolute
// path for "test" in the home path. An absolute path is returned unchanged.
func (paths *Path) Resolve(fileType FileType, path string) string {
	// absolute paths are not changed for non-hostfs file types, since hostfs is a little odd
	if filepath.IsAbs(path) {
		return path
	}

	switch fileType {
	case Home:
		return filepath.Join(paths.Home, path)
	case Config:
		return filepath.Join(paths.Config, path)
	case Data:
		return filepath.Join(paths.Data, path)
	case Logs:
		return filepath.Join(paths.Logs, path)
	default:
		panic(fmt.Sprintf("Unknown file type: %s", fileType))
	}
}

// String returns a textual representation
func (paths *Path) String() string {
	return fmt.Sprintf("Home path: [%s] Config path: [%s] Data path: [%s] Logs path: [%s]",
		paths.Home, paths.Config, paths.Data, paths.Logs)
}
