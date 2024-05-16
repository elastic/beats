// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix && !windows

package inputs

import (
	"github.com/elastic/beats/v7/filebeat/beater"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/awscloudwatch"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/awss3"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureblobstorage"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/benchmark"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/cel"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/cloudfoundry"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/gcs"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/http_endpoint"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/httpjson"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/lumberjack"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/o365audit"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/salesforce"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/shipper"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/websocket"
	"github.com/elastic/elastic-agent-libs/logp"
)

func xpackInputs(info beat.Info, log *logp.Logger, store beater.StateStore) []v2.Plugin {
	return []v2.Plugin{
		azureblobstorage.Plugin(log, store),
		cel.Plugin(log, store),
		cloudfoundry.Plugin(),
		entityanalytics.Plugin(log),
		gcs.Plugin(log, store),
		http_endpoint.Plugin(),
		httpjson.Plugin(log, store),
		o365audit.Plugin(log, store),
		awss3.Plugin(store),
		awscloudwatch.Plugin(),
		lumberjack.Plugin(),
		salesforce.Plugin(log, store),
		shipper.Plugin(log, store),
		websocket.Plugin(log, store),
		netflow.Plugin(log),
		benchmark.Plugin(),
	}
}
