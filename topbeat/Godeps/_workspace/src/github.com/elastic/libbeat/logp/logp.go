package logp

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"runtime"
	"strings"
)

// cmd line flags
var verbose *bool
var toStderr *bool
var debugSelectorsStr *string

type Logging struct {
	Selectors []string
	Files     *FileRotator
	To_syslog *bool
	To_files  *bool
	Level     string
}

func init() {
	// Adds logging specific flags: -v, -e and -d.
	verbose = flag.Bool("v", false, "Log at INFO level")
	toStderr = flag.Bool("e", false, "Log to stderr and disable syslog/file output")
	debugSelectorsStr = flag.String("d", "", "Enable certain debug selectors")
}

// Init combines the configuration from config with the command line
// flags to initialize the Logging systems. After calling this function,
// standard output is always enabled. You can make it respect the command
// line flag with a later SetStderr call.
func Init(name string, config *Logging) error {

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

	var defaultToFiles, defaultToSyslog bool
	var defaultFilePath string
	if runtime.GOOS == "windows" {
		// always disabled on windows
		defaultToSyslog = false
		defaultToFiles = true
		defaultFilePath = fmt.Sprintf("C:\\ProgramData\\%s\\Logs", name)
	} else {
		defaultToSyslog = true
		defaultToFiles = false
		defaultFilePath = fmt.Sprintf("/var/log/%s", name)
	}

	var toSyslog, toFiles bool
	if config.To_syslog != nil {
		toSyslog = *config.To_syslog
	} else {
		toSyslog = defaultToSyslog
	}
	if config.To_files != nil {
		toFiles = *config.To_files
	} else {
		toFiles = defaultToFiles
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
		return LOG_ERR, nil
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
