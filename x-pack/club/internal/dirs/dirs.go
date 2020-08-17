// Package dirs provides an unified view on disk layout for common directories.
package dirs

//go:generate godocdown -plain=false -output Readme.md

import (
	"fmt"
	"os"
	"path/filepath"
)

// Project lists common directory types an application would require for its
// internal use. Use any of the ProjectFromX functions to initialize project.
type Project struct {
	Home   string
	Config string
	Data   string
	Logs   string
}

// ProjectFromCWD initializes a Project, assuming the elastic common directory layout.
func ProjectFromCWD() (Project, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Project{}, fmt.Errorf("Failed to read working directory: %w", err)
	}

	return makeProjectFromPath(cwd), nil
}

// ProjectFrom initializes a Project directory listing, using path as the home
// directory. The home directory layout is assumed to follow the elastic common
// directory layout.
// The current working directory will be used if home is not empty.
func ProjectFrom(home string) (Project, error) {
	if home == "" {
		return ProjectFromCWD()
	}
	return makeProjectFromPath(home), nil
}

// Update overwrites paths using the list of updates. Empty entries in updates
// are ignored.
func (p Project) Update(updates Project) Project {
	if updates.Home != "" {
		p.Home = updates.Home
	}
	if updates.Config != "" {
		p.Config = updates.Config
	}
	if updates.Data != "" {
		p.Data = updates.Data
	}
	if updates.Logs != "" {
		p.Logs = updates.Logs
	}
	return p
}

func makeProjectFromPath(p string) Project {
	return Project{
		Home:   p,
		Config: p,
		Data:   filepath.Join(p, "data"),
		Logs:   filepath.Join(p, "logs"),
	}
}
