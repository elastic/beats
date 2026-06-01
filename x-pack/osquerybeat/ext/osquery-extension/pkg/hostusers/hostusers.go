// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux || darwin

package hostusers

import (
	"context"
	"strconv"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/client"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hostfs"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	elastichostusers "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/elastic_host_users"
)

const passwdFile = "/etc/passwd"

func init() {
	elastichostusers.RegisterGenerateFunc(func(ctx context.Context, queryContext table.QueryContext, log *logger.Logger, _ *client.ResilientClient) ([]elastichostusers.Result, error) {
		return getResults(ctx, queryContext, log)
	})
}

func getResults(_ context.Context, queryContext table.QueryContext, log *logger.Logger) ([]elastichostusers.Result, error) {
	fn := hostfs.GetPath(passwdFile)
	log.Infof("reading passwd for path: %s", fn)
	rows, err := hostfs.ReadPasswd(fn)
	if err != nil {
		return nil, err
	}
	results := make([]elastichostusers.Result, 0, len(rows))
	for _, m := range rows {
		uid, _ := strconv.ParseInt(m["uid"], 10, 64)
		gid, _ := strconv.ParseInt(m["gid"], 10, 64)
		uidSigned, _ := strconv.ParseInt(m["uid_signed"], 10, 64)
		gidSigned, _ := strconv.ParseInt(m["gid_signed"], 10, 64)
		results = append(results, elastichostusers.Result{
			Uid:         uid,
			Gid:         gid,
			UidSigned:   uidSigned,
			GidSigned:   gidSigned,
			Username:    m["username"],
			Description: m["description"],
			Directory:   m["directory"],
			Shell:       m["shell"],
			Uuid:        m["uuid"],
		})
	}
	return results, nil
}
