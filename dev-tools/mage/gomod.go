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

	"github.com/elastic/beats/v7/dev-tools/mage/gotool"
)

// copyModule contains a module name and the list of files or directories
// to copy recursively.
type copyModule struct {
	name        string
	filesToCopy []string
}

var (
	copyAll = []copyModule{
		copyModule{
			name: "github.com/godror/godror",
			filesToCopy: []string{
				"odpi",
			},
		},
		copyModule{
			name: "github.com/tsg/go-daemon",
			filesToCopy: []string{
				"src",
			},
		},
	}
)

// Vendor cleans up go.mod and copies the files not carried over from modules cache.
func Vendor() error {
	mod := gotool.Mod

	err := mod.Tidy()
	if err != nil {
		return err
	}

	err = mod.Vendor()
	if err != nil {
		return err
	}

	err = mod.Verify()
	if err != nil {
		return err
	}

	repo, err := GetProjectRepoInfo()
	if err != nil {
		return err
	}
	vendorFolder := filepath.Join(repo.RootDir, "vendor")

	// copy packages which require the whole tree
	for _, p := range copyAll {
		path, err := gotool.ListModuleVendorDir(p.name)
		if err != nil {
			return err
		}

		for _, f := range p.filesToCopy {
			from := filepath.Join(path, f)
			to := filepath.Join(vendorFolder, p.name, f)
			copyTask := &CopyTask{Source: from, Dest: to, DirMode: os.ModeDir | 0750}
			err = copyTask.Execute()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
