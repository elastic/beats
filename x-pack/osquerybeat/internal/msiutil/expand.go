// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package msiutil

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/command"
)

// Expand runs msiextract to extract the MSI.
func Expand(msiFile, dstDir string) error {
	output, err := command.Execute(context.Background(), "msiextract", "--directory", dstDir, msiFile)
	if err != nil {
		return fmt.Errorf("failed to run msiextract: %w (output: %s)", err, output)
	}
	return nil
}
