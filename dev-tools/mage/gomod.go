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

package mage

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/dev-tools/mage/gotool"
)

// CopyModule contains a module name and the list of files or directories
// to copy recursively.
type CopyModule struct {
	Name        string
	FilesToCopy []string
}

var (
	copyAll = []CopyModule{
		CopyModule{
			Name: "github.com/tsg/go-daemon",
			FilesToCopy: []string{
				"src",
			},
		},
		CopyModule{
			Name: "github.com/godror/godror",
			FilesToCopy: []string{
				"odpi",
			},
		},
	}
	filesToRemove = []string{
		filepath.Join("github.com", "yuin", "gopher-lua", "parse", "Makefile"),
		filepath.Join("github.com", "yuin", "gopher-lua", "parse", "parser.go.y"),
	}
)

// Vendor cleans up go.mod and copies the files not carried over from modules cache.
func Vendor() error {
	mod := gotool.Mod

	err := mod.Tidy()
	if err != nil {
		return errors.Wrap(err, "error while running go mod tidy")
	}

	err = mod.Vendor()
	if err != nil {
		return errors.Wrap(err, "error while running go mod vendor")
	}

	err = mod.Verify()
	if err != nil {
		return errors.Wrap(err, "error while running go mod verify")
	}

	repo, err := GetProjectRepoInfo()
	if err != nil {
		return errors.Wrap(err, "error while getting repository information")
	}

	vendorFolder := filepath.Join(repo.RootDir, "vendor")
	err = CopyFilesToVendor(vendorFolder, copyAll)
	if err != nil {
		return errors.Wrap(err, "error copying required files")
	}

	for _, p := range filesToRemove {
		p = filepath.Join(vendorFolder, p)
		err = os.RemoveAll(p)
		if err != nil {
			return errors.Wrapf(err, "error while removing file: %s", p)
		}
	}
	return nil
}

// CopyFilesToVendor copies packages which require the whole tree
func CopyFilesToVendor(vendorFolder string, modulesToCopy []CopyModule) error {
	for _, p := range modulesToCopy {
		path, err := gotool.ListModuleCacheDir(p.Name)
		if err != nil {
			return errors.Wrapf(err, "error while looking up cached dir of module: %s", p.Name)
		}

		for _, f := range p.FilesToCopy {
			from := filepath.Join(path, f)
			to := filepath.Join(vendorFolder, p.Name, f)
			copyTask := &CopyTask{Source: from, Dest: to, Mode: 0600, DirMode: os.ModeDir | 0750}
			err = copyTask.Execute()
			if err != nil {
				return errors.Wrapf(err, "error while copying file from %s to %s", from, to)
			}
		}
	}
	return nil
}
