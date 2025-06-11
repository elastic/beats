// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package query

import (
	"fmt"
	"net/url"

	"github.com/go-sql-driver/mysql"
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

const TLSConfigKey = "custom"

// ParseDSN tries to parse the host
func ParseDSN(mod mb.Module, host string) (mb.HostData, error) {
	// TODO: Add support for `username` and `password` as module options
	config := ConnectionDetails{}
	if err := mod.UnpackConfig(&config); err != nil {
		return mb.HostData{}, fmt.Errorf("error parsing config file: %w", err)
	}

	if config.Driver == "oracle" {
		return oracleParseDSN(config, host)
	}

	if config.Driver == "mysql" {
		return mysqlParseDSN(config, host)
	}

	sanitized := sanitize(host)
	return mb.HostData{
		URI:          host,
		SanitizedURI: sanitized,
		Host:         sanitized,
	}, nil
}

func sanitize(host string) string {
	// Host is a standard URL
	if url, err := url.Parse(host); err == nil && len(url.Host) > 0 {
		return url.Host
	}

	return "(redacted)"
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
		params.StandaloneConnection = true
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

func mysqlParseDSN(config ConnectionDetails, host string) (mb.HostData, error) {
	c, err := mysql.ParseDSN(host)

	if err != nil {
		return mb.HostData{}, fmt.Errorf("error trying to parse connection string in field 'hosts': %w", err)
	}

	sanitized := c.Addr

	if config.TLS.IsEnabled() {
		c.TLSConfig = TLSConfigKey

		tlsConfig, err := tlscommon.LoadTLSConfig(config.TLS)
		if err != nil {
			return mb.HostData{}, fmt.Errorf("could not load provided TLS configuration: %w", err)
		}

		if err := mysql.RegisterTLSConfig(TLSConfigKey, tlsConfig.ToConfig()); err != nil {
			return mb.HostData{}, fmt.Errorf("registering custom tls config failed: %w", err)
		}
	}

	return mb.HostData{
		URI:          c.FormatDSN(),
		SanitizedURI: sanitized,
		Host:         sanitized,
	}, nil
}
