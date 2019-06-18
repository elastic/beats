package dataframes

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
)

// Config is used for unpacking a common.Config.
type Config struct {
	Mode        Mode                   `config:enabled`
	Source      string                 `config:source`
	Dest        string                 `config:dest`
	Interval    string                 `config:timespan`
	Transform   map[string]interface{} `config:transform`
	CheckExists bool                   `config:"check_exists"`
	// Enable always overwrite policy mode. This required manage_ilm privileges.
	Overwrite bool `config:"overwrite"`
}

type Mode uint8

const (
	//ModeAuto enum 'auto'
	ModeAuto Mode = iota

	//ModeEnabled enum 'true'
	ModeEnabled

	//ModeDisabled enum 'false'
	ModeDisabled
)

func defaultConfig(info beat.Info) Config {
	source := fmt.Sprintf("%s-%s", info.Beat, info.Version)
	dest := fmt.Sprintf("%s-states", info.Beat)

	return Config{
		Source:    source,
		Dest:      dest,
		Interval:  "10s",
		Mode:      ModeAuto,
		Transform: map[string]interface{}{},
	}
}
