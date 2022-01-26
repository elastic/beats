// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tables

import (
	"context"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hostfs"
)

const (
	groupFile = "/etc/group"
)

func HostGroupsColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.BigIntColumn("gid"),
		table.BigIntColumn("gid_signed"),
		table.TextColumn("groupname"),
	}
}

func GetHostGroupsGenerateFunc() table.GenerateFunc {
	fn := hostfs.GetPath(groupFile)
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		return hostfs.ReadGroup(fn)
	}
}
