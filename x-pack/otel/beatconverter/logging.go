// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatconverter

import (
	"fmt"
	"strings"
)

func getOTelLogLevel(level string) (string, error) {
	switch strings.ToLower(level) {
	case "debug":
		return "DEBUG", nil
	case "info":
		return "INFO", nil
	case "warning":
		return "WARN", nil
	case "error", "critical":
		return "ERROR", nil
	default:
		return "", fmt.Errorf("unrecognized level: %s", level)
	}
}
