// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beats

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
)

const (
	// args: data path, pipeline name, application name
	logFileFormat = "%s/logs/%s/%s-json.log"
	// args: data path, install path, pipeline name, application name
	logFileFormatWin = "%s\\logs\\%s\\%s-json.log"

	// args: pipeline name, application name
	mbEndpointFileFormat = "unix:///tmp/elastic-agent/%s/%s/%s.sock"
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
		return fmt.Sprintf(logFileFormatWin, paths.Home(), pipelineID, program)
	}

	return fmt.Sprintf(logFileFormat, paths.Home(), pipelineID, program)
}
