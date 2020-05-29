// Package project provides utility for reviewdog execution based on project
// config.
package project

import "gopkg.in/yaml.v2"

// Config represents reviewdog config.
type Config struct {
	Runner map[string]*Runner
}

// Runner represents config for a runner.
type Runner struct {
	// Runner command. (e.g. `golint ./...`)
	Cmd string
	// tool name in review comment. (e.g. `golint`)
	Name string
	// errorformat name. (e.g. `checkstyle`)
	Format string
	// errorformat. (e.g. `%f:%l:%c:%m`, `%-G%.%#`)
	Errorformat []string
	// Report Level for this runner. ("info", "warning", "error")
	Level string
}

// Parse parses reviewdog config in yaml format.
func Parse(yml []byte) (*Config, error) {
	out := &Config{}
	if err := yaml.Unmarshal(yml, out); err != nil {
		return nil, err
	}
	// Insert `Name` field if it's empty.
	for name, runner := range out.Runner {
		if runner.Name == "" {
			runner.Name = name
		}
	}
	return out, nil
}
