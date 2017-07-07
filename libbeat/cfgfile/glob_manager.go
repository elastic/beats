package cfgfile

import (
	"os"
	"path/filepath"
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
	files             []*cfgfile
}

type cfgfile struct {
	name    string
	path    string
	enabled bool
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
		g.files = append(g.files, &cfgfile{
			name:    strings.TrimSuffix(filepath.Base(path), g.enabledExtension),
			enabled: true,
			path:    path,
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
		g.files = append(g.files, &cfgfile{
			name:    strings.TrimSuffix(filepath.Base(path), g.enabledExtension+g.disabledExtension),
			enabled: false,
			path:    path,
		})
	}

	return nil
}

// ListEnabled conf files
func (g *GlobManager) ListEnabled() []string {
	var names []string
	for _, file := range g.files {
		if file.enabled {
			names = append(names, file.name)
		}
	}

	return names
}

// ListDisabled conf files
func (g *GlobManager) ListDisabled() []string {
	var names []string
	for _, file := range g.files {
		if !file.enabled {
			names = append(names, file.name)
		}
	}

	return names
}

// Enabled returns true if given conf file is enabled
func (g *GlobManager) Enabled(name string) bool {
	for _, file := range g.files {
		if name == file.name {
			return file.enabled
		}
	}
	return false
}

// Exists return true if the given conf exists (enabled or disabled)
func (g *GlobManager) Exists(name string) bool {
	for _, file := range g.files {
		if name == file.name {
			return true
		}
	}
	return false
}

// Enable given conf file, does nothing if it's enabled already
func (g *GlobManager) Enable(name string) error {
	for _, file := range g.files {
		if name == file.name {
			if !file.enabled {
				newPath := strings.TrimSuffix(file.path, g.disabledExtension)
				if err := os.Rename(file.path, newPath); err != nil {
					return errors.Wrap(err, "enable failed")
				}
				file.enabled = true
				file.path = newPath
			}
			return nil
		}
	}

	return errors.Errorf("module %s not found", name)
}

// Disable given conf file, does nothing if it's disabled already
func (g *GlobManager) Disable(name string) error {
	for _, file := range g.files {
		if name == file.name {
			if file.enabled {
				newPath := file.path + g.disabledExtension
				if err := os.Rename(file.path, newPath); err != nil {
					return errors.Wrap(err, "disable failed")
				}
				file.enabled = false
				file.path = newPath
			}
			return nil
		}
	}

	return errors.Errorf("module %s not found", name)
}
