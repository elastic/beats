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

	"github.com/elastic/beats/v8/dev-tools/mage/gotool"
)

// CopyModule contains a module name and the list of files or directories
// to copy recursively.
type CopyModule struct {
	Name        string
	FilesToCopy []string
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
