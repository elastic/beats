// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build darwin

package fileanalysis

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/encoding"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/command"
)

func executeStderr(ctx context.Context, name string, arg ...string) (out string, err error) {
	cmd := exec.CommandContext(ctx, name, arg...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return "", err
	}

	return stderr.String(), nil
}

func generate(log *logger.Logger) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		var results []map[string]string

		pathConstraint, exists := queryContext.Constraints["path"]
		if !exists {
			return results, nil
		}

		var path *string
		for _, constraint := range pathConstraint.Constraints {
			constraintCopy := constraint
			path = &constraintCopy.Expression
			break
		}

		if path == nil {
			return results, nil
		}

		// Validate and sanitize the input path
		stat, err := os.Stat(*path)
		if err != nil {
			log.Errorf("Error stating file path '%s': %v", *path, err)
			return results, fmt.Errorf("error accessing file path %s: %w", *path, err)
		}

		if !stat.Mode().IsRegular() {
			log.Errorf("Path is not a regular file: %s", *path)
			return results, fmt.Errorf("invalid path: %s", *path)
		}

		sys, ok := stat.Sys().(*syscall.Stat_t)
		if !ok {
			log.Errorf("Unable to convert stat.Sys() to *syscall.Stat_t for path: %s", *path)
			return results, fmt.Errorf("unable to convert stat.Sys() to *syscall.Stat_t")
		}

		// Execute macOS commands
		fileType, err := command.Execute(ctx, "file", *path)
		if err != nil {
			log.Warningf("Error running 'file' command: %v", err)
		}

		dependencies, err := command.Execute(ctx, "otool", "-L", *path)
		if err != nil {
			log.Warningf("Error running 'otool' command: %v", err)
		}

		symbols, err := command.Execute(ctx, "nm", *path)
		if err != nil {
			log.Warningf("Error running 'nm' command: %v", err)
		}

		stringsOutput, err := command.Execute(ctx, "strings", "-a", *path)
		if err != nil {
			log.Warningf("Error running 'strings' command: %v", err)
		}

		// Execute macOS codesign command and capture stderr for output
		codeSign, err := executeStderr(ctx, "codesign", "-dvvv", *path)
		if err != nil {
			log.Warningf("Error running 'codesign' command: %v", err)
		}

		// Create fileAnalysis struct
		analysis := &fileAnalysis{
			Path:         *path,
			Mode:         fmt.Sprintf("%o", stat.Mode().Perm()),
			UID:          int64(sys.Uid),
			GID:          int64(sys.Gid),
			Size:         stat.Size(),
			Mtime:        stat.ModTime().Unix(),
			FileType:     strings.TrimSpace(fileType),
			CodeSign:     strings.TrimSpace(codeSign),
			Dependencies: strings.TrimSpace(dependencies),
			Symbols:      strings.TrimSpace(symbols),
			Strings:      strings.TrimSpace(stringsOutput),
		}

		// Convert to map using encoding
		result, err := encoding.MarshalToMapWithFlags(analysis, encoding.EncodingFlagUseNumbersZeroValues)
		if err != nil {
			log.Errorf("Error marshaling file analysis: %v", err)
			return results, fmt.Errorf("error marshaling file analysis: %w", err)
		}

		results = append(results, result)
		return results, nil
	}
}
