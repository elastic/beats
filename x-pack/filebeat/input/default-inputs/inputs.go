// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package inputs

import (
	ossinputs "github.com/elastic/beats/v7/filebeat/input/default-inputs"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/logp"
)

func Init(info beat.Info, log *logp.Logger, store statestore.States) []v2.Plugin {
	return append(
		xpackInputs(info, log, store),
		ossinputs.Init(info, log, store)...,
	)
}
