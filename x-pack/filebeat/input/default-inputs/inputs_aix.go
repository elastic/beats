// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package inputs

import (
	"github.com/elastic/beats/v7/filebeat/beater"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/awss3"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/http_endpoint"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/httpjson"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/o365audit"
	"github.com/elastic/elastic-agent-libs/logp"
)

func xpackInputs(info beat.Info, log *logp.Logger, store beater.StateStore) []v2.Plugin {
	return []v2.Plugin{
		http_endpoint.Plugin(),
		httpjson.Plugin(log, store),
		o365audit.Plugin(log, store),
		awss3.Plugin(store),
	}
}
