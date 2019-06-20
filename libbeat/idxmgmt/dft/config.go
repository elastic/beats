package dft

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
)

// Config is used for unpacking a common.Config.
type Config struct {
	Mode        Mode               `config:enabled`
	Transform   DataFrameTransform `config:transform`
	CheckExists bool               `config:"check_exists"`
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

func DefaultConfig(info beat.Info) Config {
	name := fmt.Sprintf("%s-states", info.Beat)
	source := fmt.Sprintf("%s-%s-*", info.Beat, info.Version)
	dest := fmt.Sprintf("%s-states", info.Beat)

	return Config{
		Mode: ModeAuto,
		Transform: DataFrameTransform{
			Name:     name,
			Source:   source,
			Dest:     dest,
			Interval: "10s",
		},
	}
}
