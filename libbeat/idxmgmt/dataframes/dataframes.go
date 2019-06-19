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
	Manager(h ClientHandler) Manager
}

// Manager uses a ClientHandler to install a policy.
type Manager interface {
	Enabled() (bool, error)

	EnsureDataframes() error
}

// DefaultSupport configures a new default ILM support implementation.
func DefaultSupport(log *logp.Logger, info beat.Info, config *common.Config) (Supporter, error) {
	cfg := defaultConfig(info)
	if config != nil {
		if err := config.Unpack(&cfg); err != nil {
			return nil, err
		}
	}

	// TODO: IMPLEMENT THIS
	//if cfg.Mode == ModeDisabled {
	//	return NewNoopSupport(info, config)
	//}

	return StdSupport(log, info, config)
}

func StdSupport(log *logp.Logger, info beat.Info, config *common.Config) (Supporter, error) {
	if log == nil {
		log = logp.NewLogger("dataframe")
	} else {
		log = log.Named("dataframe")
	}

	cfg := defaultConfig(info)
	if config != nil {
		if err := config.Unpack(&cfg); err != nil {
			return nil, err
		}
	}

	return NewStdSupport(log, cfg.Mode, cfg.Source, cfg.Dest, cfg.Interval, cfg.Transform), nil
}
