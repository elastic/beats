// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package warn

import (
	"fmt"
	"io"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

const message = "The Elastic Agent is currently in Experimental and should not be used in production"

// LogNotGA warns the users in the log that the Elastic Agent is not GA.
func LogNotGA(log *logger.Logger) {
	log.Info(message)
}

// PrintNotGA writes to the received writer that the Agent is not GA.
func PrintNotGA(output io.Writer) {
	fmt.Fprintln(output, message)
}
