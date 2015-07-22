package logp

import (
	"flag"
	"io/ioutil"
	"log"
	"strings"
)

// cmd line flags
var verbose *bool
var toStderr *bool
var debugSelectorsStr *string

type Logging struct {
	Selectors []string
	Files     *FileRotator
}

// Init combines the configuration from config with the command line
// flags to initialize the Logging systems. After calling this function,
// standard output is always enabled. You can make it respect the command
// line flag with a later SetStderr call.
func Init(config *Logging) {

	logLevel := LOG_ERR
	if *verbose {
		logLevel = LOG_INFO
	}

	debugSelectors := []string{}
	if len(*debugSelectorsStr) > 0 {
		debugSelectors = strings.Split(*debugSelectorsStr, ",")
		logLevel = LOG_DEBUG
	}

	LogInit(Priority(logLevel), "", !*toStderr, true, debugSelectors)
	if len(debugSelectors) > 0 {
		config.Selectors = debugSelectors
	}

	if IsDebug("stdlog") {
		// disable standard logging by default (this is sometimes
		// used by libraries and we don't want their logs to spam ours)
		log.SetOutput(ioutil.Discard)
	}
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
