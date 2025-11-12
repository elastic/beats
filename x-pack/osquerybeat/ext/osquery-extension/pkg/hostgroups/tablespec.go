// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hostgroups

import (
	"context"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/encoding"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hostfs"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tablespec"
)

const (
	groupFile = "/etc/group"
)

func TableSpec() *tablespec.TableSpec {
	columns, err := encoding.GenerateColumnDefinitions(hostGroup{})
	if err != nil {
		panic(err)
	}
	return tablespec.NewTableSpec(
		"host_groups",
		"Host groups information from /etc/group",
		[]string{"linux", "darwin"},
		columns,
		generate,
	)
}

func generate(log *logger.Logger) table.GenerateFunc {
	fn := hostfs.GetPath(groupFile)
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		log.Infof("reading group for path: %s", fn)
		return hostfs.ReadGroup(fn)
	}
}
