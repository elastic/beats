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

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	elasticfileanalysis "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/elastic_file_analysis"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/command"
)

func init() {
	elasticfileanalysis.RegisterGenerateFunc(getResults)
}

func executeStderr(ctx context.Context, name string, arg ...string) (out string, err error) {
	cmd := exec.CommandContext(ctx, name, arg...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err = cmd.Run(); err != nil {
		return "", err
	}
	return stderr.String(), nil
}

func getResults(ctx context.Context, queryContext table.QueryContext, log *logger.Logger) ([]elasticfileanalysis.Result, error) {
	var results []elasticfileanalysis.Result

	pathConstraint, exists := queryContext.Constraints["path"]
	if !exists {
		return results, nil
	}

	var path *string
	for _, constraint := range pathConstraint.Constraints {
		c := constraint.Expression
		path = &c
		break
	}
	if path == nil {
		return results, nil
	}

	stat, err := os.Stat(*path)
	if err != nil {
		log.Errorf("Error stating file path '%s': %v", *path, err)
		return nil, fmt.Errorf("error accessing file path %s: %w", *path, err)
	}
	if !stat.Mode().IsRegular() {
		log.Errorf("Path is not a regular file: %s", *path)
		return nil, fmt.Errorf("invalid path: %s", *path)
	}

	sys, ok := stat.Sys().(*syscall.Stat_t)
	if !ok {
		log.Errorf("Unable to convert stat.Sys() to *syscall.Stat_t for path: %s", *path)
		return nil, fmt.Errorf("unable to convert stat.Sys() to *syscall.Stat_t")
	}

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
	codeSign, err := executeStderr(ctx, "codesign", "-dvvv", *path)
	if err != nil {
		log.Warningf("Error running 'codesign' command: %v", err)
	}

	results = append(results, elasticfileanalysis.Result{
		Path:         *path,
		Mode:         fmt.Sprintf("%o", stat.Mode().Perm()),
		Uid:          int64(sys.Uid),
		Gid:          int64(sys.Gid),
		Size:         stat.Size(),
		Mtime:        stat.ModTime().Unix(),
		FileType:     strings.TrimSpace(fileType),
		CodeSign:     strings.TrimSpace(codeSign),
		Dependencies: strings.TrimSpace(dependencies),
		Symbols:      strings.TrimSpace(symbols),
		Strings:      strings.TrimSpace(stringsOutput),
	})
	return results, nil
}
