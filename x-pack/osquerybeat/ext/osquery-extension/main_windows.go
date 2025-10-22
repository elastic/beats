// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package main

import (
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/browserhistory"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

func RegisterTables(server *osquery.ExtensionManagerServer, log func(m string, kvs ...any)) {
	server.RegisterPlugin(table.NewPlugin("browser_history", browserhistory.GetColumns(), browserhistory.GetGenerateFunc(log)))
}
