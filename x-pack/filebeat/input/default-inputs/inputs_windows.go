// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package inputs

import (
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/awscloudwatch"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/awss3"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureblobstorage"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureeventhub"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/benchmark"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/cel"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/cloudfoundry"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/etw"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/gcs"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/http_endpoint"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/httpjson"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/lumberjack"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/o365audit"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/salesforce"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/streaming"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

func xpackInputs(info beat.Info, log *logp.Logger, store statestore.States, path *paths.Path) []v2.Plugin {
	return []v2.Plugin{
		azureblobstorage.Plugin(log, store),
		azureeventhub.Plugin(log),
		cel.Plugin(log, store),
		cloudfoundry.Plugin(),
		entityanalytics.Plugin(log, path),
		gcs.Plugin(log, store),
		http_endpoint.Plugin(log),
		httpjson.Plugin(log, store),
		o365audit.Plugin(log, store),
		awss3.Plugin(log, store, path),
		awscloudwatch.Plugin(log, store),
		lumberjack.Plugin(log),
		etw.Plugin(),
		streaming.Plugin(log, store),
		streaming.PluginWebsocketAlias(log, store),
		netflow.Plugin(log),
		salesforce.Plugin(log, store),
		benchmark.Plugin(),
	}
}
