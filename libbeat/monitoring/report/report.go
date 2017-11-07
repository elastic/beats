package report

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

type config struct {
	// allow for maximum one reporter being configured
	Reporter common.ConfigNamespace `config:",inline"`
}

type Reporter interface {
	Stop()
}

type ReporterFactory func(beat.Info, *common.Config) (Reporter, error)

var (
	defaultConfig = config{}

	reportFactories = map[string]ReporterFactory{}
)

func RegisterReporterFactory(name string, f ReporterFactory) {
	if reportFactories[name] != nil {
		panic(fmt.Sprintf("Reporter '%v' already registered", name))
	}
	reportFactories[name] = f
}

func New(
	beat beat.Info,
	cfg *common.Config,
	outputs common.ConfigNamespace,
) (Reporter, error) {
	name, cfg, err := getReporterConfig(cfg, outputs)
	if err != nil {
		return nil, err
	}

	f := reportFactories[name]
	if f == nil {
		return nil, fmt.Errorf("unknown reporter type '%v'", name)
	}

	return f(beat, cfg)
}

func getReporterConfig(
	cfg *common.Config,
	outputs common.ConfigNamespace,
) (string, *common.Config, error) {
	cfg = collectSubObject(cfg)
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return "", nil, err
	}

	// load reporter from `monitoring` section and optionally
	// merge with output settings
	if config.Reporter.IsSet() {
		name := config.Reporter.Name()
		rc := config.Reporter.Config()

		// merge reporter config with output config if both are present
		if outCfg := outputs.Config(); outputs.Name() == name && outCfg != nil {
			// require monitoring to not configure any hosts if output is configured:
			hosts := struct {
				Hosts []string `config:"hosts"`
			}{}
			rc.Unpack(&hosts)

			if len(hosts.Hosts) > 0 {
				pathMonHosts := rc.PathOf("hosts")
				pathOutHost := outCfg.PathOf("hosts")
				err := fmt.Errorf("'%v' and '%v' are configured", pathMonHosts, pathOutHost)
				return "", nil, err
			}

			merged, err := common.MergeConfigs(outCfg, rc)
			if err != nil {
				return "", nil, err
			}
			rc = merged
		}

		return name, rc, nil
	}

	// find output also available for reporting telemetry.
	if outputs.IsSet() {
		name := outputs.Name()
		if reportFactories[name] != nil {
			return name, outputs.Config(), nil
		}
	}

	return "", nil, errors.New("No monitoring reporter configured")
}

func collectSubObject(cfg *common.Config) *common.Config {
	out := common.NewConfig()
	for _, field := range cfg.GetFields() {
		if obj, err := cfg.Child(field, -1); err == nil {
			// on error field is no object, but primitive value -> ignore
			out.SetChild(field, -1, obj)
			continue
		}
	}
	return out
}
