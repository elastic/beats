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
	StartTime   time.Time `struct:"start_timestamp"`
	LogDateTime string    `struct:"timestamp"`
}

func ParseCursor(cfg *config, cursor *state, log *logp.Logger) (qr string, err error) {
	ctxTmpl := mapstr.M{
		"var":    nil,
		"cursor": nil,
	}

	if cursor.LogDateTime != "" {
		ctxTmpl["cursor"] = mapstr.M{"logdate": cursor.LogDateTime}
		qr, err = cfg.Query.Value.Execute(ctxTmpl, nil, log)
		if err != nil {
			return "", err
		}
		return qr, nil
	}

	defaultTmpl := cfg.Query.Default
	ctxTmpl["var"] = mapstr.M{"initial_interval": time.Now().Add(-cfg.InitialInterval).Format(time.RFC3339)}
	qr, err = cfg.Query.Default.Execute(ctxTmpl, defaultTmpl, log)
	if err != nil {
		return "", err
	}

	return qr, nil
}
