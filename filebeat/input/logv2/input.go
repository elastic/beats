package logv2

import (
	"fmt"

	"github.com/elastic/beats/v7/filebeat/channel"
	v1 "github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/filebeat/input/filestream"
	loginput "github.com/elastic/beats/v7/filebeat/input/log"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
)

const pluginName = "log"

func init() {
	// Register an input V1, that's used by the log input
	if err := v1.Register(pluginName, newV1Input); err != nil {
		panic(err)
	}
}

// newV1Input creates a new log input
func newV1Input(
	cfg *config.C,
	outlet channel.Connector,
	context v1.Context,
	logger *logp.Logger,
) (v1.Input, error) {
	if ok, _ := cfg.Has("be_filestream", -1); ok {
		beFilestream, err := cfg.Bool("be_filestream", -1)
		if err != nil {
			return nil, fmt.Errorf("newV1Input: cannot parse 'be_filestream': %w", err)
		}

		if beFilestream {
			if err := cfg.SetString("type", -1, "filestream"); err != nil {
				return nil, fmt.Errorf("cannot set 'type': %w", err)
			}

			return nil, v2.ErrUnknownInput
		}
	}

	inp, err := loginput.NewInput(cfg, outlet, context, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create log input: %w", err)
	}

	return inp, err
}

// PluginV2 proxies the call to filestream's Plugin function
func PluginV2(logger *logp.Logger, store statestore.States) v2.Plugin {
	// The InputManager for Filestream input is from an internal package, so we
	// cannot instantiate it directly here. To circumvent that, we instantiate
	// the whole Filestream Plugin
	filestreamPlugin := filestream.Plugin(logger, store)

	m := manager{
		next: filestreamPlugin.Manager,
	}
	filestreamPlugin.Manager = m

	p := v2.Plugin{
		Name:      pluginName,
		Stability: feature.Experimental,
		Info:      "log input running filestream",
		Doc:       "Log input running Filestream input",
		Manager:   m,
	}
	return p
}

type manager struct {
	next v2.InputManager
}

func (m manager) Init(grp unison.Group) error {
	return m.next.Init(grp)
}

func (m manager) Create(cfg *config.C) (v2.Input, error) {
	if ok, _ := cfg.Has("be_filestream", -1); ok {
		beFilestream, err := cfg.Bool("be_filestream", -1)
		if err != nil {
			return nil, fmt.Errorf("manager.Create: cannot parse 'be_filestream': %w", err)
		}

		if beFilestream {
			if err := cfg.SetString("type", -1, "filestream"); err != nil {
				return nil, fmt.Errorf("cannot set 'type': %w", err)
			}

			if err := cfg.SetBool("take_over.enabled", -1, true); err != nil {
				return nil, fmt.Errorf("cannot set 'take_over.enabled': %w", err)
			}

			return m.next.Create(cfg)
		}
	}

	return nil, v2.ErrUnknownInput
}
