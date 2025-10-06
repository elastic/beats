// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"context"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	_ "github.com/mattn/go-sqlite3"
	"github.com/osquery/osquery-go/plugin/table"
)

// GetTable discovers all users, all profiles, all browsers automatically
func GetTable(ctx context.Context, queryContext table.QueryContext, log func(m string, kvs ...any)) ([]map[string]string, error) {
	results := make([]map[string]string, 0)

	userPaths := discoverUsers(log)

	var merr *multierror.Error
	for _, browser := range defaultBrowsers {
		parser, found := browserParsers[browser]
		if !found || parser == nil {
			continue
		}
		for _, userPath := range userPaths {
			pathPattern := getBrowserPath(browser)
			if pathPattern == "" {
				continue
			}
			bresults, err := parser(ctx, queryContext, browser, filepath.Join(userPath, pathPattern), log)
			if err != nil {
				merr = multierror.Append(merr, err)
				continue
			}
			results = append(results, bresults...)
		}
	}

	return results, merr.ErrorOrNil()
}
