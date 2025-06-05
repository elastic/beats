// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package utils

import (
	"os"
	"strconv"
)

func strDefault(a, defaults string) string {
	if a == "" {
		return defaults
	}
	return a
}

func boolDefault(a string, defaults bool) bool {
	b, err := strconv.ParseBool(a)

	if err != nil {
		return defaults
	}

	return b
}

func floatDefault(a string, defaults float64) float64 {
	b, err := strconv.ParseFloat(a, 64)

	if err != nil {
		return defaults
	}

	return b
}

func intDefault(a string, defaults int) int {
	b, err := strconv.ParseInt(a, 10, 64)

	if err != nil {
		return defaults
	}

	return int(b)
}

// GetStrenv environment variable, if not supplied returns the default value
func GetStrenv(name, defaultValue string) string {
	return strDefault(os.Getenv(name), defaultValue)
}

// GetBooleanEnvParam environment variable, if not supplied returns the default value
func GetBooleanEnvParam(name string, defaultValue bool) bool {
	return boolDefault(os.Getenv(name), defaultValue)
}

// GetIntEnvParam environment variable, if not supplied returns the default value
func GetFloatEnvParam(name string, defaultValue float64) float64 {
	return floatDefault(os.Getenv(name), defaultValue)
}

func GetIntEnvParam(name string, defaultValue int) int {
	return intDefault(os.Getenv(name), defaultValue)
}
