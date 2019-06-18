package dataframes

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// SupportFactory is used to define a policy type to be used.
type SupportFactory func(*logp.Logger, beat.Info, *common.Config) (Supporter, error)

type Supporter interface {
	Mode() Mode
}

func DefaultSupport(log *logp.Logger, info beat.Info, config *common.Config) (Supporter, error) {
}
