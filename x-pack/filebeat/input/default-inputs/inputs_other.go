// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package inputs

import (
	"github.com/menderesk/beats/v7/filebeat/beater"
	v2 "github.com/menderesk/beats/v7/filebeat/input/v2"
	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/awscloudwatch"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/awss3"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/cloudfoundry"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/http_endpoint"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/httpjson"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/o365audit"
)

func xpackInputs(info beat.Info, log *logp.Logger, store beater.StateStore) []v2.Plugin {
	return []v2.Plugin{
		cloudfoundry.Plugin(),
		http_endpoint.Plugin(),
		httpjson.Plugin(log, store),
		o365audit.Plugin(log, store),
		awss3.Plugin(store),
		awscloudwatch.Plugin(store),
	}
}
