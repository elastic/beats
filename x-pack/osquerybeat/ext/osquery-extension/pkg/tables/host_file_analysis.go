// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tables

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/command"
	"github.com/osquery/osquery-go/plugin/table"
)

func FileAnalysisColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("path"),
		table.TextColumn("mode"),
		table.TextColumn("uid"),
		table.TextColumn("gid"),
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
			path = &constraint.Expression
			break
		}
		if path == nil {
			return results, nil
		}

		stat, err := os.Stat(*path)
		if err != nil {
			return results, nil
		}

		sys, ok := stat.Sys().(*syscall.Stat_t)
		if !ok {
			return results, fmt.Errorf("unable to convert stat.Sys() to *syscall.Stat_t")
		}

		mode := fmt.Sprintf("%o", stat.Mode().Perm())
		uid := strconv.Itoa(int(sys.Uid))
		gid := strconv.Itoa(int(sys.Gid))
		size := strconv.Itoa(int(stat.Size()))
		mtime := strconv.FormatInt(stat.ModTime().Unix(), 10)

		// Execute macOS commands
		fileType, _ := command.Execute(ctx, "file", *path)
		codeSign, _ := command.Execute(ctx, "codesign", "-dvvv", *path)
		dependencies, _ := command.Execute(ctx, "otool", "-L", *path)
		symbols, _ := command.Execute(ctx, "nm", *path)
		stringsOutput, _ := command.Execute(ctx, "strings", "-a", *path)

		// Convert outputs to strings
		fileTypeStr := strings.TrimSpace(string(fileType))
		codeSignStr := strings.TrimSpace(string(codeSign))
		dependenciesStr := strings.TrimSpace(string(dependencies))
		symbolsStr := strings.TrimSpace(string(symbols))
		stringsStr := strings.TrimSpace(string(stringsOutput))

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
