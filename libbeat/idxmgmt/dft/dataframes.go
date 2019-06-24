package dft

import (
	"fmt"
	"path"

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

type DataFrameTransform struct {
	Name               string        `config:"name"`
	Pivot              common.MapStr `config:"pivot"`
	DestMappings       common.MapStr `config:"dest.mappings"`
	SourceMetaIdx      string        `config:"source.meta.index"`
	SourceMetaMappings common.MapStr `config:"source.meta.mappings"`
	SourceIdx          string        `config:"source.index"`
	DestIdx            string        `config:"dest.index"`
	Interval           string        `config:"timespan"`
	Pipeline           Pipeline      `config:"pipeline"`
}
type Pipeline struct {
	ID          string          `config:"id"`
	Description string          `config:"description"`
	Processors  []common.MapStr `config:"processors"`
}

func (t *DataFrameTransform) path() string {
	return path.Join(esDFTPath, t.Name)
}

// Manager uses a ClientHandler to install a policy.
type Manager interface {
	Enabled() (bool, error)

	EnsureDataframes() error
}

// DefaultSupport configures a new default ILM support implementation.
func DefaultSupport(log *logp.Logger, info beat.Info, config *common.Config) (Supporter, error) {
	cfg := DefaultConfig(info)
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

	cfg := DefaultConfig(info)
	if config != nil {
		if err := config.Unpack(&cfg); err != nil {
			return nil, err
		}
	}

	u, _ := config.Child("transforms", 0)
	fmt.Printf("UNPACKED INTO %v from %v\n", cfg.Transforms[0], u.GetFields())

	return NewStdSupport(log, cfg.Mode, cfg.Transforms), nil
}
