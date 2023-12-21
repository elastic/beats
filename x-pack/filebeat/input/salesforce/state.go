// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type state struct {
	Object       dateTimeCursor `json:"object,omitempty"`
	EventLogFile dateTimeCursor `json:"event_log_file,omitempty"`
}

type dateTimeCursor struct {
	FirstEventTime string `struct:"first_event_time,omitempty"`
	LastEventTime  string `struct:"last_event_time,omitempty"`
}

func parseCursor(initialInterval *time.Duration, cfg *QueryConfig, cursor mapstr.M, log *logp.Logger) (string, error) {
	ctxTmpl := mapstr.M{"cursor": nil}

	if cursor != nil {
		ctxTmpl["cursor"] = cursor
		qr, err := cfg.Value.Execute(ctxTmpl, nil, log)
		if err != nil {
			return "", err
		}
		return qr, nil
	}

	return cfg.Default.Execute(ctxTmpl, nil, log)
}
