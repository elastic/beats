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

package cfgfile

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

// GlobManager allows to manage a directory of conf files. Using a glob pattern
// to match them, this object will allow to switch their state between enabled
// and disabled
type GlobManager struct {
	glob              string
	enabledExtension  string
	disabledExtension string
	files             []*CfgFile
}

type CfgFile struct {
	Name    string
	Path    string
	Enabled bool
}

// NewGlobManager takes a glob and enabled/disabled extensions and returns a GlobManager object.
// Parameters:
//  - glob - matching conf files (ie: modules.d/*.yml)
//  - enabledExtension - extension for enabled confs, must match the glob (ie: .yml)
//  - disabledExtension - extension to append for disabled confs (ie: .disabled)
func NewGlobManager(glob, enabledExtension, disabledExtension string) (*GlobManager, error) {
	if !strings.HasSuffix(glob, enabledExtension) {
		return nil, errors.New("Glob should have the enabledExtension as suffix")
	}

	g := &GlobManager{
		glob:              glob,
		enabledExtension:  enabledExtension,
		disabledExtension: disabledExtension,
	}
	if err := g.load(); err != nil {
		return nil, err
	}
	return g, nil
}

func (g *GlobManager) load() error {
	// empty previous data
	g.files = nil

	// Load enabled
	watcher := NewGlobWatcher(g.glob)
	files, _, err := watcher.Scan()
	if err != nil {
		return err
	}

	for _, path := range files {
		// Trim cfg file name
		g.files = append(g.files, &CfgFile{
			Name:    strings.TrimSuffix(filepath.Base(path), g.enabledExtension),
			Enabled: true,
			Path:    path,
		})
	}

	// Load disabled
	watcher = NewGlobWatcher(g.glob + g.disabledExtension)
	files, _, err = watcher.Scan()
	if err != nil {
		return err
	}

	for _, path := range files {
		// Trim cfg file name
		g.files = append(g.files, &CfgFile{
			Name:    strings.TrimSuffix(filepath.Base(path), g.enabledExtension+g.disabledExtension),
			Enabled: false,
			Path:    path,
		})
	}

	return nil
}

// ListEnabled conf files
func (g *GlobManager) ListEnabled() []*CfgFile {
	var enabled []*CfgFile
	for _, file := range g.files {
		if file.Enabled {
			enabled = append(enabled, file)
		}
	}

	sort.Sort(byCfgFileDisplayNames(enabled))
	return enabled
}

// ListDisabled conf files
func (g *GlobManager) ListDisabled() []*CfgFile {
	var disabled []*CfgFile
	for _, file := range g.files {
		if !file.Enabled {
			disabled = append(disabled, file)
		}
	}

	sort.Sort(byCfgFileDisplayNames(disabled))
	return disabled
}

// Enabled returns true if given conf file is enabled
func (g *GlobManager) Enabled(name string) bool {
	for _, file := range g.files {
		if name == file.Name {
			return file.Enabled
		}
	}
	return false
}

// Exists return true if the given conf exists (enabled or disabled)
func (g *GlobManager) Exists(name string) bool {
	for _, file := range g.files {
		if name == file.Name {
			return true
		}
	}
	return false
}

// Enable given conf file, does nothing if it's enabled already
func (g *GlobManager) Enable(name string) error {
	for _, file := range g.files {
		if name == file.Name {
			if !file.Enabled {
				newPath := strings.TrimSuffix(file.Path, g.disabledExtension)
				if err := os.Rename(file.Path, newPath); err != nil {
					return errors.Wrap(err, "enable failed")
				}
				file.Enabled = true
				file.Path = newPath
			}
			return nil
		}
	}

	return errors.Errorf("module %s not found", name)
}

// Disable given conf file, does nothing if it's disabled already
func (g *GlobManager) Disable(name string) error {
	for _, file := range g.files {
		if name == file.Name {
			if file.Enabled {
				newPath := file.Path + g.disabledExtension
				if err := os.Rename(file.Path, newPath); err != nil {
					return errors.Wrap(err, "disable failed")
				}
				file.Enabled = false
				file.Path = newPath
			}
			return nil
		}
	}

	return errors.Errorf("module %s not found", name)
}

// For sorting config files in the desired order, so variants will
// show up after the default, e.g. elasticsearch-xpack will show up
// after elasticsearch.
type byCfgFileDisplayNames []*CfgFile

func (s byCfgFileDisplayNames) Len() int {
	return len(s)
}

func (s byCfgFileDisplayNames) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byCfgFileDisplayNames) Less(i, j int) bool {
	namei := s[i].Name
	namej := s[j].Name

	if strings.HasPrefix(namei, namej) {
		// namei starts with namej, so namei is longer and we want it to come after namej
		return false
	} else if strings.HasPrefix(namej, namei) {
		// namej starts with namei, so namej is longer and we want it to come after namei
		return true
	} else {
		return namei < namej
	}
}
