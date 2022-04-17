// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tables

import (
	"context"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/menderesk/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hostfs"
)

const (
	passwdFile = "/etc/passwd"
)

func HostUsersColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.BigIntColumn("uid"),
		table.BigIntColumn("gid"),
		table.BigIntColumn("uid_signed"),
		table.BigIntColumn("gid_signed"),
		table.TextColumn("username"),
		table.TextColumn("description"),
		table.TextColumn("directory"),
		table.TextColumn("shell"),
		table.TextColumn("uuid"),
	}
}

func GetHostUsersGenerateFunc() table.GenerateFunc {
	fn := hostfs.GetPath(passwdFile)
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		return hostfs.ReadPasswd(fn)
	}
}
