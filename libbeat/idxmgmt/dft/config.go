package dft

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/libbeat/beat"
)

// Config is used for unpacking a common.Config.
type Config struct {
	Mode        Mode                  `config:enabled`
	Transforms  []*DataFrameTransform `config:transforms`
	CheckExists bool                  `config:"check_exists"`
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
	majorV := strings.Split(info.Version, ".")[0]
	majorXX := fmt.Sprintf("%s.x.x", majorV)
	majorXAny := fmt.Sprintf("%s.*", majorV)

	name := fmt.Sprintf("%s-states", info.Beat)
	metaIdx := fmt.Sprintf("%s-%s-meta", info.Beat, majorXX)
	source := fmt.Sprintf("%s-%s", info.Beat, majorXAny)
	dest := fmt.Sprintf("%s-states-%s", info.Beat, majorXX)

	return Config{
		Mode: ModeAuto,
		Transforms: []*DataFrameTransform{
			{
				Name:          name,
				SourceMetaIdx: metaIdx,
				SourceIdx:     source,
				DestIdx:       dest,
				DestMappings:  common.MapStr{},
				Interval:      "10s",
			},
		},
	}
}
