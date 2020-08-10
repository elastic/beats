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
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	monitoringConfig "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/config"
)

const httpPlusPrefix = "http+"

// Monitor is a monitoring interface providing information about the way
// how beat is monitored
type Monitor struct {
	operatingSystem string
	config          *monitoringConfig.MonitoringConfig
	installPath     string
}

// NewMonitor creates a beats monitor.
func NewMonitor(downloadConfig *artifact.Config, monitoringCfg *monitoringConfig.MonitoringConfig) *Monitor {
	if monitoringCfg == nil {
		monitoringCfg = monitoringConfig.DefaultConfig()
	}

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
		b.config = cfg.Settings.MonitoringConfig
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

// WatchLogs returns true if monitoring is enabled and monitor should watch logs.
func (b *Monitor) WatchLogs() bool { return b.config.Enabled && b.config.MonitorLogs }

// WatchMetrics returns true if monitoring is enabled and monitor should watch metrics.
func (b *Monitor) WatchMetrics() bool { return b.config.Enabled && b.config.MonitorMetrics }

func (b *Monitor) generateMonitoringEndpoint(process, pipelineID string) string {
	return getMonitoringEndpoint(process, b.operatingSystem, pipelineID)
}

func (b *Monitor) generateLoggingFile(process, pipelineID string) string {
	return getLoggingFile(process, b.operatingSystem, b.installPath, pipelineID)
}

func (b *Monitor) generateLoggingPath(process, pipelineID string) string {
	return filepath.Dir(b.generateLoggingFile(process, pipelineID))

}

// EnrichArgs enriches arguments provided to application, in order to enable
// monitoring
func (b *Monitor) EnrichArgs(process, pipelineID string, args []string, isSidecar bool) []string {
	appendix := make([]string, 0, 7)

	monitoringEndpoint := b.generateMonitoringEndpoint(process, pipelineID)
	if monitoringEndpoint != "" {
		endpoint := monitoringEndpoint
		if isSidecar {
			endpoint += "_monitor"
		}
		appendix = append(appendix,
			"-E", "http.enabled=true",
			"-E", "http.host="+endpoint,
		)
	}

	loggingPath := b.generateLoggingPath(process, pipelineID)
	if loggingPath != "" {
		logFile := process
		if isSidecar {
			logFile += "_monitor"
		}
		logFile = fmt.Sprintf("%s-json.log", logFile)
		appendix = append(appendix,
			"-E", "logging.json=true",
			"-E", "logging.ecs=true",
			"-E", "logging.files.path="+loggingPath,
			"-E", "logging.files.name="+logFile,
			"-E", "logging.files.keepfiles=7",
			"-E", "logging.files.permission=0640",
			"-E", "logging.files.interval=1h",
		)
	}

	return append(args, appendix...)
}

// Cleanup removes
func (b *Monitor) Cleanup(process, pipelineID string) error {
	// do not cleanup logs, they might not be all processed
	drop := b.monitoringDrop(process, pipelineID)
	if drop == "" {
		return nil
	}

	return os.RemoveAll(drop)
}

// Prepare executes steps in order for monitoring to work correctly
func (b *Monitor) Prepare(process, pipelineID string, uid, gid int) error {
	drops := []string{b.generateLoggingPath(process, pipelineID)}
	if drop := b.monitoringDrop(process, pipelineID); drop != "" {
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
// application is not monitorable
func (b *Monitor) LogPath(process, pipelineID string) string {
	if !b.WatchLogs() {
		return ""
	}

	return b.generateLoggingFile(process, pipelineID)
}

// MetricsPath describes a location where application exposes metrics
// collectable by metricbeat.
func (b *Monitor) MetricsPath(process, pipelineID string) string {
	if !b.WatchMetrics() {
		return ""
	}

	return b.generateMonitoringEndpoint(process, pipelineID)
}

// MetricsPathPrefixed return metrics path prefixed with http+ prefix.
func (b *Monitor) MetricsPathPrefixed(process, pipelineID string) string {
	return httpPlusPrefix + b.MetricsPath(process, pipelineID)
}

func (b *Monitor) monitoringDrop(process, pipelineID string) string {
	return monitoringDrop(b.generateMonitoringEndpoint(process, pipelineID))
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

	if strings.HasPrefix(path, httpPlusPrefix) {
		path = strings.TrimPrefix(path, httpPlusPrefix)
	}

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
