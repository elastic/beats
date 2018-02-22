package configure

import (
	"flag"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// CLI flags for configuring logging.
var (
	verbose        bool
	toStderr       bool
	debugSelectors []string
)

func init() {
	flag.BoolVar(&verbose, "v", false, "Log at INFO level")
	flag.BoolVar(&toStderr, "e", false, "Log to stderr and disable syslog/file output")
	common.StringArrVarFlag(nil, &debugSelectors, "d", "Enable certain debug selectors")
}

// Logging builds a logp.Config based on the given common.Config and the specified
// CLI flags.
func Logging(beatName string, cfg *common.Config) error {
	config := logp.DefaultConfig()
	config.Beat = beatName
	if cfg != nil {
		if err := cfg.Unpack(&config); err != nil {
			return err
		}
	}

	applyFlags(&config)
	return logp.Configure(config)
}

func applyFlags(cfg *logp.Config) {
	if toStderr {
		cfg.ToStderr = true
	}
	if cfg.Level > logp.InfoLevel && verbose {
		cfg.Level = logp.InfoLevel
	}
	for _, selectors := range debugSelectors {
		cfg.Selectors = append(cfg.Selectors, strings.Split(selectors, ",")...)
	}

	// Elevate level if selectors are specified on the CLI.
	if len(debugSelectors) > 0 {
		cfg.Level = logp.DebugLevel
	}
}
