// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beats

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact"
)

const httpPlusPrefix = "http+"

// Monitor is a monitoring interface providing information about the way
// how beat is monitored
type Monitor struct {
	pipelineID string

	process            string
	monitoringEndpoint string
	loggingPath        string

	monitorLogs    bool
	monitorMetrics bool
}

// NewMonitor creates a beats monitor.
func NewMonitor(process, pipelineID string, downloadConfig *artifact.Config, monitorLogs, monitorMetrics bool) *Monitor {
	var monitoringEndpoint, loggingPath string

	if monitorMetrics {
		monitoringEndpoint = getMonitoringEndpoint(process, downloadConfig.OS(), pipelineID)
	}
	if monitorLogs {
		loggingPath = getLoggingFileDirectory(downloadConfig.InstallPath, downloadConfig.OS(), pipelineID)
	}

	return &Monitor{
		pipelineID:         pipelineID,
		process:            process,
		monitoringEndpoint: monitoringEndpoint,
		loggingPath:        loggingPath,
		monitorLogs:        monitorLogs,
		monitorMetrics:     monitorMetrics,
	}
}

// EnrichArgs enriches arguments provided to application, in order to enable
// monitoring
func (b *Monitor) EnrichArgs(args []string) []string {
	appendix := make([]string, 0, 7)

	if b.monitoringEndpoint != "" {
		appendix = append(appendix,
			"-E", "http.enabled=true",
			"-E", "http.host="+b.monitoringEndpoint,
		)
	}

	if b.loggingPath != "" {
		appendix = append(appendix,
			"-E", "logging.files.path="+b.loggingPath,
			"-E", "logging.files.name="+b.process,
			"-E", "logging.files.keepfiles=7",
			"-E", "logging.files.permission=0644",
			"-E", "logging.files.interval=1h",
		)
	}

	return append(args, appendix...)
}

// Cleanup removes
func (b *Monitor) Cleanup() error {
	// do not cleanup logs, they might not be all processed
	drop := b.monitoringDrop()
	if drop == "" {
		return nil
	}

	return os.RemoveAll(drop)
}

// Prepare executes steps in order for monitoring to work correctly
func (b *Monitor) Prepare(uid, gid int) error {
	drops := []string{b.loggingPath}
	if drop := b.monitoringDrop(); drop != "" {
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

		if err := os.Chown(drop, uid, gid); err != nil {
			return err
		}
	}

	return nil
}

// LogPath describes a path where application stores logs. Empty if
// application is not monitorable
func (b *Monitor) LogPath() string {
	if !b.monitorLogs {
		return ""
	}

	return b.loggingPath
}

// MetricsPath describes a location where application exposes metrics
// collectable by metricbeat.
func (b *Monitor) MetricsPath() string {
	if !b.monitorMetrics {
		return ""
	}

	return b.monitoringEndpoint
}

// MetricsPathPrefixed return metrics path prefixed with http+ prefix.
func (b *Monitor) MetricsPathPrefixed() string {
	return httpPlusPrefix + b.MetricsPath()
}

func (b *Monitor) monitoringDrop() string {
	return monitoringDrop(b.monitoringEndpoint)
}

func monitoringDrop(path string) (drop string) {
	defer func() {
		if drop != "" {
			drop = filepath.Dir(drop)
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
