package report

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
)

type config struct {
	// allow for maximum one reporter being configured
	Reporter common.ConfigNamespace `config:",inline"`
}

type Reporter interface {
	Stop()
}

type ReporterFactory func(common.BeatInfo, *common.Config) (Reporter, error)

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
	beat common.BeatInfo,
	cfg *common.Config,
	outputs map[string]*common.Config,
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
	outputs map[string]*common.Config,
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
		if outCfg := outputs[name]; outCfg != nil {
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
	// Fail if multiple potential reporters have been found
	var found string
	for name := range outputs {
		if reportFactories[name] == nil {
			continue
		}

		if found != "" {
			err := fmt.Errorf("multiple potential monitoring reporters found (for example %v and %v)", found, name)
			return "", nil, err
		}
		found = name
	}

	if found == "" {
		return "", nil, errors.New("No monitoring reporter configured")
	}

	return found, outputs[found], nil
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
