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
}

// Init combines the configuration from config with the command line
// flags to initialize the Logging systems. After calling this function,
// standard output is always enabled. You can make it respect the command
// line flag with a later SetStderr call.
func Init(name string, config *Logging) error {

	logLevel := LOG_ERR
	if *verbose {
		logLevel = LOG_INFO
	}

	debugSelectors := []string{}
	if len(*debugSelectorsStr) > 0 {
		debugSelectors = strings.Split(*debugSelectorsStr, ",")
		logLevel = LOG_DEBUG
	}

	var defaultToFiles, defaultToSyslog bool
	if runtime.GOOS == "windows" {
		// always disabled on windows
		defaultToSyslog = false
		defaultToFiles = true
	} else {
		defaultToSyslog = true
		defaultToFiles = false
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
	toSyslog = toSyslog && !*toStderr
	toFiles = toFiles && !*toStderr

	LogInit(Priority(logLevel), "", toSyslog, true, debugSelectors)
	if len(debugSelectors) > 0 {
		config.Selectors = debugSelectors
	}

	if toFiles {
		if config.Files == nil {
			if runtime.GOOS == "windows" {
				config.Files = &FileRotator{
					Path: fmt.Sprintf("C:\\ProgramData\\%s\\Logs", name),
					Name: name,
				}
			} else {
				config.Files = &FileRotator{
					Path: fmt.Sprintf("/var/log/%s", name),
					Name: name,
				}
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
		Info("Startup successful, disable stdout logging")
		SetToStderr(false, "")
	}
}

// Adds logging specific flags to the flag set. The taken flags are
// -v, -e and -d.
func CmdLineFlags(flags *flag.FlagSet) {
	verbose = flags.Bool("v", false, "Log at INFO level")
	toStderr = flags.Bool("e", false, "Output to stdout and disable syslog/file output")
	debugSelectorsStr = flags.String("d", "", "Enable certain debug selectors")
}
