// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beats

import (
	"fmt"
	"path/filepath"
)

const (
	// args: pipeline name, application name
	logFileFormat = "/var/log/elastic-agent/%s/%s"
	// args: install path, pipeline name, application name
	logFileFormatWin = "%s\\logs\\elastic-agent\\%s\\%s"

	// args: pipeline name, application name
	mbEndpointFileFormat = "unix:///var/run/elastic-agent/%s/%s/%s.sock"
	// args: pipeline name, application name
	mbEndpointFileFormatWin = `npipe:///%s-%s`
)

func getMonitoringEndpoint(program, operatingSystem, pipelineID string) string {
	if operatingSystem == "windows" {
		return fmt.Sprintf(mbEndpointFileFormatWin, pipelineID, program)
	}

	return fmt.Sprintf(mbEndpointFileFormat, pipelineID, program, program)
}

func getLoggingFile(program, operatingSystem, installPath, pipelineID string) string {
	if operatingSystem == "windows" {
		return fmt.Sprintf(logFileFormatWin, installPath, pipelineID, program)
	}

	return fmt.Sprintf(logFileFormat, pipelineID, program)
}

func getLoggingFileDirectory(installPath, operatingSystem, pipelineID string) string {
	return filepath.Base(getLoggingFile("program", operatingSystem, installPath, pipelineID))
}
