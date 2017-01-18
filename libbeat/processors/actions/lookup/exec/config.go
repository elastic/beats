package exec

import (
	"time"

	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/processors/actions/lookup/lutool"
)

type execLookupConfig struct {
	Key    []string           `config:"key"`
	Cache  lutool.CacheConfig `config:",inline"`
	Runner execRunnerConfig   `config:",inline"`
}

type execRunnerConfig struct {
	Run        *fmtstr.EventFormatString `config:"run"          validate:"required"`
	Timeout    time.Duration             `config:"timeout"      validate:"min=0"`
	User       string                    `config:"user"`
	Chroot     string                    `config:"chroot"`
	WorkingDir string                    `config:"working_directory"`
	Env        []string                  `config:"env"`

	// XXX: Querying group information support is lacking in golang runtime.
	//      Do support `group` config for now.
	// Group      string        `config:"group"`
}

var (
	defaultConfig = execLookupConfig{
		Key:   nil,
		Cache: lutool.DefaultCacheConfig,
		Runner: execRunnerConfig{
			Timeout:    60 * time.Second,
			User:       "",
			Chroot:     "",
			WorkingDir: "",
			Env:        nil,
		},
	}
)
