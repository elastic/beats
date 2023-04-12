// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tables

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/command"
)

func ExecuteStderr(ctx context.Context, name string, arg ...string) (out string, err error) {
	cmd := exec.CommandContext(ctx, name, arg...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return stderr.String(), err
	}

	return stderr.String(), nil
}

func FileAnalysisColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("path"),
		table.TextColumn("mode"),
		table.BigIntColumn("uid"),
		table.BigIntColumn("gid"),
		table.TextColumn("size"),
		table.TextColumn("mtime"),
		table.TextColumn("file_type"),
		table.TextColumn("code_sign"),
		table.TextColumn("dependencies"),
		table.TextColumn("symbols"),
		table.TextColumn("strings"),
	}
}

func GetFileAnalysisGenerateFunc() table.GenerateFunc {
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
		if err != nil || !stat.Mode().IsRegular() {
			return results, fmt.Errorf("invalid path: %s", *path)
		}

		var uid, gid string = "0", "0"

		if runtime.GOOS == "darwin" {
			sys, ok := stat.Sys().(*syscall.Stat_t)
			if !ok {
				return results, fmt.Errorf("unable to convert stat.Sys() to *syscall.Stat_t")
			}
			uid = strconv.FormatUint(uint64(sys.Uid), 10)
			gid = strconv.FormatUint(uint64(sys.Gid), 10)
		}

		mode := fmt.Sprintf("%o", stat.Mode().Perm())

		size := strconv.FormatUint(uint64(stat.Size()), 10)
		mtime := strconv.FormatInt(stat.ModTime().Unix(), 10)

		// Execute macOS commands
		fileType, _ := command.Execute(ctx, "file", *path)
		dependencies, _ := command.Execute(ctx, "otool", "-L", *path)
		symbols, _ := command.Execute(ctx, "nm", *path)
		stringsOutput, _ := command.Execute(ctx, "strings", "-a", *path)

		// Execute macOS codesign command and capture stderr for output
		codeSign, err := ExecuteStderr(ctx, "codesign", "-dvvv", *path)
		if err != nil {
			log.Println("Error running codesign command:", err)
		}

		// Convert outputs to strings
		fileTypeStr := strings.TrimSpace(fileType)
		codeSignStr := strings.TrimSpace(codeSign)
		dependenciesStr := strings.TrimSpace(dependencies)
		symbolsStr := strings.TrimSpace(symbols)
		stringsStr := strings.TrimSpace(stringsOutput)

		results = append(results, map[string]string{
			"path":         *path,
			"mode":         mode,
			"uid":          uid,
			"gid":          gid,
			"size":         size,
			"mtime":        mtime,
			"file_type":    fileTypeStr,
			"code_sign":    codeSignStr,
			"dependencies": dependenciesStr,
			"symbols":      symbolsStr,
			"strings":      stringsStr,
		})

		return results, nil
	}
}
