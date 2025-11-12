// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tables

import (
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/browserhistory"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/fileanalysis"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hostgroups"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hostprocesses"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hostusers"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tablespec"
)

func Init() {
	tablespec.MustRegister(fileanalysis.TableSpec())
	tablespec.MustRegister(hostgroups.TableSpec())
	tablespec.MustRegister(hostprocesses.TableSpec())
	tablespec.MustRegister(hostusers.TableSpec())
	tablespec.MustRegister(browserhistory.TableSpec())
}
