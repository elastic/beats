package dirs

import (
	"fmt"
	"os"
	"path/filepath"
)

type Project struct {
	Home   string
	Config string
	Data   string
	Logs   string
}

func ProjectFromCWD() (Project, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Project{}, fmt.Errorf("Failed to read working directory: %w", err)
	}

	return makeProjectFromPath(cwd), nil
}

func ProjectFrom(path string) (Project, error) {
	if path == "" {
		return ProjectFromCWD()
	}
	return makeProjectFromPath(path), nil
}

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
