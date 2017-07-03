package logp

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/paths"
)

var (
	// cmd line flags
	verbose           *bool
	toStderr          *bool
	debugSelectorsStr *string

	// Beat start time
	startTime time.Time
)

type Logging struct {
	Selectors []string
	Files     *FileRotator
	ToSyslog  *bool `config:"to_syslog"`
	ToFiles   *bool `config:"to_files"`
	JSON      bool  `config:"json"`
	Level     string
	Metrics   LoggingMetricsConfig `config:"metrics"`
}

type LoggingMetricsConfig struct {
	Enabled *bool          `config:"enabled"`
	Period  *time.Duration `config:"period" validate:"nonzero,min=0s"`
}

var (
	defaultMetricsPeriod = 30 * time.Second
)

func init() {
	startTime = time.Now()

	// Adds logging specific flags: -v, -e and -d.
	verbose = flag.Bool("v", false, "Log at INFO level")
	toStderr = flag.Bool("e", false, "Log to stderr and disable syslog/file output")
	debugSelectorsStr = flag.String("d", "", "Enable certain debug selectors")
}

func HandleFlags(name string) error {
	level := _log.level
	if *verbose {
		if LOG_INFO > level {
			level = LOG_INFO
		}
	}

	selectors := strings.Split(*debugSelectorsStr, ",")
	debugSelectors, debugAll := parseSelectors(selectors)
	if debugAll || len(debugSelectors) > 0 {
		level = LOG_DEBUG
	}

	// flags are handled before config file is read => log to stderr for now
	_log.level = level
	_log.toStderr = true
	_log.logger = log.New(os.Stderr, name, stderrLogFlags)
	_log.selectors = debugSelectors
	_log.debugAllSelectors = debugAll

	return nil
}

// Init combines the configuration from config with the command line
// flags to initialize the Logging systems. After calling this function,
// standard output is always enabled. You can make it respect the command
// line flag with a later SetStderr call.
func Init(name string, config *Logging) error {
	// reset settings from HandleFlags
	_log = logger{
		JSON: config.JSON,
	}

	logLevel, err := getLogLevel(config)
	if err != nil {
		return err
	}

	if *verbose {
		if LOG_INFO > logLevel {
			logLevel = LOG_INFO
		}
	}

	debugSelectors := config.Selectors
	if logLevel == LOG_DEBUG {
		if len(debugSelectors) == 0 {
			debugSelectors = []string{"*"}
		}
	}
	if len(*debugSelectorsStr) > 0 {
		debugSelectors = strings.Split(*debugSelectorsStr, ",")
		logLevel = LOG_DEBUG
	}

	// default log location is in the logs path
	defaultFilePath := paths.Resolve(paths.Logs, "")

	var toSyslog, toFiles bool
	if config.ToSyslog != nil {
		toSyslog = *config.ToSyslog
	} else {
		toSyslog = false
	}
	if config.ToFiles != nil {
		toFiles = *config.ToFiles
	} else {
		toFiles = true
	}

	// toStderr disables logging to syslog/files
	if *toStderr {
		toSyslog = false
		toFiles = false
	}

	LogInit(Priority(logLevel), "", toSyslog, true, debugSelectors)
	if len(debugSelectors) > 0 {
		config.Selectors = debugSelectors
	}

	if toFiles {
		if config.Files == nil {
			config.Files = &FileRotator{
				Path: defaultFilePath,
				Name: name,
			}
		} else {
			if config.Files.Path == "" {
				config.Files.Path = defaultFilePath
			}

			if config.Files.Name == "" {
				config.Files.Name = name
			}
		}

		err := SetToFile(true, config.Files)
		if err != nil {
			return err
		}
	}

	if IsDebug("stdlog") {
		// disable standard logging by default (this is sometimes
		// used by libraries and we don't want their logs to spam ours)
		log.SetOutput(ioutil.Discard)
	}

	// Disable stderr logging if requested by cmdline flag
	SetStderr()

	go logMetrics(&config.Metrics)

	return nil
}

func SetStderr() {
	if !*toStderr {
		SetToStderr(false, "")
		Debug("log", "Disable stderr logging")
	}
}

func getLogLevel(config *Logging) (Priority, error) {
	if config == nil || config.Level == "" {
		return LOG_INFO, nil
	}

	levels := map[string]Priority{
		"critical": LOG_CRIT,
		"error":    LOG_ERR,
		"warning":  LOG_WARNING,
		"info":     LOG_INFO,
		"debug":    LOG_DEBUG,
	}

	level, ok := levels[strings.ToLower(config.Level)]
	if !ok {
		return 0, fmt.Errorf("unknown log level: %v", config.Level)
	}
	return level, nil
}
