// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || darwin

package hostgroups

import (
	"context"
	"strconv"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hooks"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hostfs"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	elastichostgroups "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/elastic_host_groups"
	hostgroupsview "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/views/generated/host_groups"
)

const groupFile = "/etc/group"

func init() {
	elastichostgroups.RegisterGenerateFunc(getResults)
	hostgroupsview.RegisterHooksFunc(func(hm *hooks.HookManager) {
		hostgroupsview.RegisterDefaultViewHook(hm)
	})
}

func getResults(ctx context.Context, queryContext table.QueryContext, log *logger.Logger) ([]elastichostgroups.Result, error) {
	fn := hostfs.GetPath(groupFile)
	log.Infof("reading group for path: %s", fn)
	rows, err := hostfs.ReadGroup(fn)
	if err != nil {
		return nil, err
	}
	results := make([]elastichostgroups.Result, 0, len(rows))
	for _, m := range rows {
		gid, _ := strconv.ParseInt(m["gid"], 10, 64)
		gidSigned, _ := strconv.ParseInt(m["gid_signed"], 10, 64)
		results = append(results, elastichostgroups.Result{
			Gid:       gid,
			GidSigned: gidSigned,
			Groupname: m["groupname"],
		})
	}
	return results, nil
}
