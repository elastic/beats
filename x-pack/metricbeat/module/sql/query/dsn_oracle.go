// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package query

import (
	"fmt"

	"github.com/godror/godror"
	"github.com/godror/godror/dsn"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

// ConnectionDetails contains all possible data that can be used to create a connection with
// an Oracle db
type ConnectionDetails struct {
	Username string            `config:"username"`
	Password string            `config:"password"`
	Driver   string            `config:"driver"`
	TLS      *tlscommon.Config `config:"ssl"`
}

func oracleParseDSN(config ConnectionDetails, host string) (mb.HostData, error) {
	params, err := godror.ParseDSN(host)
	if err != nil {
		return mb.HostData{}, fmt.Errorf("error trying to parse connection string in field 'hosts': %w", err)
	}
	if params.Username == "" {
		params.Username = config.Username
	}
	if params.Password.Secret() == "" {
		params.StandaloneConnection = godror.Bool(true)
		params.Password = dsn.NewPassword(config.Password)
	}
	return mb.HostData{
		URI:          params.StringWithPassword(),
		SanitizedURI: params.ConnectString,
		Host:         params.String(),
		User:         params.Username,
		Password:     params.Password.Secret(),
	}, nil
}
