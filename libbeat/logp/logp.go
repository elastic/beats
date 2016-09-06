package logp

import (
	"expvar"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
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

func init() {
	startTime = time.Now()
}

type Logging struct {
	Selectors []string
	Files     *FileRotator
	ToSyslog  *bool `config:"to_syslog"`
	ToFiles   *bool `config:"to_files"`
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

	go logExpvars(&config.Metrics)

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

// snapshotMap recursively walks expvar Maps and records their integer expvars
// in a separate flat map.
func snapshotMap(varsMap map[string]int64, path string, mp *expvar.Map) {
	mp.Do(func(kv expvar.KeyValue) {
		switch kv.Value.(type) {
		case *expvar.Int:
			varsMap[path+"."+kv.Key], _ = strconv.ParseInt(kv.Value.String(), 10, 64)
		case *expvar.Map:
			snapshotMap(varsMap, path+"."+kv.Key, kv.Value.(*expvar.Map))
		}
	})
}

// snapshotExpvars iterates through all the defined expvars, and for the vars
// that are integers it snapshots the name and value in a separate (flat) map.
func snapshotExpvars(varsMap map[string]int64) {
	expvar.Do(func(kv expvar.KeyValue) {
		switch kv.Value.(type) {
		case *expvar.Int:
			varsMap[kv.Key], _ = strconv.ParseInt(kv.Value.String(), 10, 64)
		case *expvar.Map:
			snapshotMap(varsMap, kv.Key, kv.Value.(*expvar.Map))
		}
	})
}

// buildMetricsOutput makes the delta between vals and prevVals and builds
// a printable string with the non-zero deltas.
func buildMetricsOutput(prevVals map[string]int64, vals map[string]int64) string {
	metrics := ""
	for k, v := range vals {
		delta := v - prevVals[k]
		if delta != 0 {
			metrics = fmt.Sprintf("%s %s=%d", metrics, k, delta)
		}
	}
	return metrics
}

// logExpvars logs at Info level the integer expvars that have changed in the
// last interval. For each expvar, the delta from the beginning of the interval
// is logged.
func logExpvars(metricsCfg *LoggingMetricsConfig) {
	if metricsCfg.Enabled != nil && *metricsCfg.Enabled == false {
		Info("Metrics logging disabled")
		return
	}
	if metricsCfg.Period == nil {
		metricsCfg.Period = &defaultMetricsPeriod
	}
	Info("Metrics logging every %s", metricsCfg.Period)

	ticker := time.NewTicker(*metricsCfg.Period)
	prevVals := map[string]int64{}
	for {
		<-ticker.C
		vals := map[string]int64{}
		snapshotExpvars(vals)
		metrics := buildMetricsOutput(prevVals, vals)
		prevVals = vals
		if len(metrics) > 0 {
			Info("Non-zero metrics in the last %s:%s", metricsCfg.Period, metrics)
		} else {
			Info("No non-zero metrics in the last %s", metricsCfg.Period)
		}
	}
}

func LogTotalExpvars(cfg *Logging) {
	if cfg.Metrics.Enabled != nil && *cfg.Metrics.Enabled == false {
		return
	}
	vals := map[string]int64{}
	prevVals := map[string]int64{}
	snapshotExpvars(vals)
	metrics := buildMetricsOutput(prevVals, vals)
	Info("Total non-zero values: %s", metrics)
	Info("Uptime: %s", time.Now().Sub(startTime))
}
