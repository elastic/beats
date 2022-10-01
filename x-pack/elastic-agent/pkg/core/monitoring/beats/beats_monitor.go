// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beats

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	monitoringConfig "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/config"
)

const httpPlusPrefix = "http+"
const defaultMonitoringNamespace = "default"

// Monitor implements the monitoring.Monitor interface providing information
// about beats.
type Monitor struct {
	operatingSystem string
	config          *monitoringConfig.MonitoringConfig
	installPath     string
}

// NewMonitor creates a beats monitor.
func NewMonitor(downloadConfig *artifact.Config, monitoringCfg *monitoringConfig.MonitoringConfig, logMetrics bool) *Monitor {
	if monitoringCfg == nil {
		monitoringCfg = monitoringConfig.DefaultConfig()
		monitoringCfg.Pprof = &monitoringConfig.PprofConfig{Enabled: false}
		monitoringCfg.HTTP.Buffer = &monitoringConfig.BufferConfig{Enabled: false}
	}
	monitoringCfg.LogMetrics = logMetrics

	return &Monitor{
		operatingSystem: downloadConfig.OS(),
		installPath:     downloadConfig.InstallPath,
		config:          monitoringCfg,
	}
}

// Reload reloads state of the monitoring based on config.
func (b *Monitor) Reload(rawConfig *config.Config) error {
	cfg := configuration.DefaultConfiguration()
	if err := rawConfig.Unpack(&cfg); err != nil {
		return err
	}

	if cfg == nil || cfg.Settings == nil || cfg.Settings.MonitoringConfig == nil {
		b.config = monitoringConfig.DefaultConfig()
	} else {
		if cfg.Settings.MonitoringConfig.Pprof == nil {
			cfg.Settings.MonitoringConfig.Pprof = b.config.Pprof
		}
		if cfg.Settings.MonitoringConfig.HTTP.Buffer == nil {
			cfg.Settings.MonitoringConfig.HTTP.Buffer = b.config.HTTP.Buffer
		}
		b.config = cfg.Settings.MonitoringConfig
		logMetrics := true
		if cfg.Settings.LoggingConfig != nil {
			logMetrics = cfg.Settings.LoggingConfig.Metrics.Enabled
		}
		b.config.LogMetrics = logMetrics
	}

	return nil
}

// Close disables monitoring
func (b *Monitor) Close() {
	b.config.Enabled = false
	b.config.MonitorMetrics = false
	b.config.MonitorLogs = false
}

// IsMonitoringEnabled returns true if monitoring is enabled.
func (b *Monitor) IsMonitoringEnabled() bool { return b.config.Enabled }

// MonitoringNamespace returns monitoring namespace configured.
func (b *Monitor) MonitoringNamespace() string {
	if b.config.Namespace == "" {
		return defaultMonitoringNamespace
	}
	return b.config.Namespace
}

// WatchLogs returns true if monitoring is enabled and monitor should watch logs.
func (b *Monitor) WatchLogs() bool { return b.config.Enabled && b.config.MonitorLogs }

// WatchMetrics returns true if monitoring is enabled and monitor should watch metrics.
func (b *Monitor) WatchMetrics() bool { return b.config.Enabled && b.config.MonitorMetrics }

func (b *Monitor) generateMonitoringEndpoint(spec program.Spec, pipelineID string) string {
	return MonitoringEndpoint(spec, b.operatingSystem, pipelineID)
}

func (b *Monitor) generateLoggingFile(spec program.Spec, pipelineID string) string {
	return getLoggingFile(spec, b.operatingSystem, b.installPath, pipelineID)
}

func (b *Monitor) generateLoggingPath(spec program.Spec, pipelineID string) string {
	return filepath.Dir(b.generateLoggingFile(spec, pipelineID))
}

func (b *Monitor) ownLoggingPath(spec program.Spec) bool {
	// if the spec file defines a custom log path then agent will not take ownership of the logging path
	_, ok := spec.LogPaths[b.operatingSystem]
	return !ok
}

// EnrichArgs enriches arguments provided to application, in order to enable
// monitoring
func (b *Monitor) EnrichArgs(spec program.Spec, pipelineID string, args []string, isSidecar bool) []string {
	appendix := make([]string, 0, 7)

	monitoringEndpoint := b.generateMonitoringEndpoint(spec, pipelineID)
	if monitoringEndpoint != "" {
		endpoint := monitoringEndpoint
		if isSidecar {
			endpoint += "_monitor"
		}
		appendix = append(appendix,
			"-E", "http.enabled=true",
			"-E", "http.host="+endpoint,
		)
		if b.config.Pprof != nil && b.config.Pprof.Enabled {
			appendix = append(appendix,
				"-E", "http.pprof.enabled=true",
			)
		}
		if b.config.HTTP.Buffer != nil && b.config.HTTP.Buffer.Enabled {
			appendix = append(appendix,
				"-E", "http.buffer.enabled=true",
			)
		}
	}

	loggingPath := b.generateLoggingPath(spec, pipelineID)
	if loggingPath != "" {
		logFile := spec.Cmd
		if isSidecar {
			logFile += "_monitor"
		}
		logFile = fmt.Sprintf("%s", logFile)
		appendix = append(appendix,
			"-E", "logging.files.path="+loggingPath,
			"-E", "logging.files.name="+logFile,
			"-E", "logging.files.keepfiles=7",
			"-E", "logging.files.permission=0640",
			"-E", "logging.files.interval=1h",
		)

		if !b.config.LogMetrics {
			appendix = append(appendix,
				"-E", "logging.metrics.enabled=false",
			)
		}
	}

	return append(args, appendix...)
}

// Cleanup removes
func (b *Monitor) Cleanup(spec program.Spec, pipelineID string) error {
	// do not cleanup logs, they might not be all processed
	drop := b.monitoringDrop(spec, pipelineID)
	if drop == "" {
		return nil
	}

	return os.RemoveAll(drop)
}

// Prepare executes steps in order for monitoring to work correctly
func (b *Monitor) Prepare(spec program.Spec, pipelineID string, uid, gid int) error {
	if !b.ownLoggingPath(spec) {
		// spec file passes a log path; so its up to the application to ensure the
		// path exists and the write permissions are set so Elastic Agent can read it
		return nil
	}

	drops := []string{b.generateLoggingPath(spec, pipelineID)}
	if drop := b.monitoringDrop(spec, pipelineID); drop != "" {
		drops = append(drops, drop)
	}

	for _, drop := range drops {
		if drop == "" {
			continue
		}

		_, err := os.Stat(drop)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}

			// create
			if err := os.MkdirAll(drop, 0775); err != nil {
				return err
			}
		}

		if err := changeOwner(drop, uid, gid); err != nil {
			return err
		}
	}

	return nil
}

// LogPath describes a path where application stores logs. Empty if
// application is not monitorable.
func (b *Monitor) LogPath(spec program.Spec, pipelineID string) string {
	if !b.WatchLogs() {
		return ""
	}

	return b.generateLoggingFile(spec, pipelineID)
}

// MetricsPath describes a location where application exposes metrics
// collectable by metricbeat.
func (b *Monitor) MetricsPath(spec program.Spec, pipelineID string) string {
	if !b.WatchMetrics() {
		return ""
	}

	return b.generateMonitoringEndpoint(spec, pipelineID)
}

// MetricsPathPrefixed return metrics path prefixed with http+ prefix.
func (b *Monitor) MetricsPathPrefixed(spec program.Spec, pipelineID string) string {
	return httpPlusPrefix + b.MetricsPath(spec, pipelineID)
}

func (b *Monitor) monitoringDrop(spec program.Spec, pipelineID string) string {
	return monitoringDrop(b.generateMonitoringEndpoint(spec, pipelineID))
}

func monitoringDrop(path string) (drop string) {
	defer func() {
		if drop != "" {
			// Dir call changes separator to the one used in OS
			// '/var/lib' -> '\var\lib\' on windows
			baseLen := len(filepath.Dir(drop))
			drop = drop[:baseLen]
		}
	}()

	if strings.Contains(path, "localhost") {
		return ""
	}

	path = strings.TrimPrefix(path, httpPlusPrefix)

	// npipe is virtual without a drop
	if isNpipe(path) {
		return ""
	}

	if isWindowsPath(path) {
		return path
	}

	u, _ := url.Parse(path)
	if u == nil || (u.Scheme != "" && u.Scheme != "file" && u.Scheme != "unix") {
		return ""
	}

	if u.Scheme == "file" {
		return strings.TrimPrefix(path, "file://")
	}

	if u.Scheme == "unix" {
		return strings.TrimPrefix(path, "unix://")
	}

	return path
}

func isNpipe(path string) bool {
	return strings.HasPrefix(path, "npipe") || strings.HasPrefix(path, `\\.\pipe\`)
}

func isWindowsPath(path string) bool {
	if len(path) < 4 {
		return false
	}
	return unicode.IsLetter(rune(path[0])) && path[1] == ':'
}

func changeOwner(path string, uid, gid int) error {
	if runtime.GOOS == "windows" {
		// on windows it always returns the syscall.EWINDOWS error, wrapped in *PathError
		return nil
	}

	return os.Chown(path, uid, gid)
}
