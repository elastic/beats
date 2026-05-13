// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package activedirectory

import (
	"crypto/tls"
	"errors"
	"net"
	"net/url"
	"time"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
	"github.com/elastic/entcollect"
	ecad "github.com/elastic/entcollect/provider/ad"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider"
)

func init() {
	err := provider.RegisterMinimalStateProvider(Name, minimalProvider)
	if err != nil {
		panic(err)
	}
}

func minimalProvider(cfg *config.C, log *logp.Logger) (entcollect.Provider, time.Duration, time.Duration, error) {
	// localConf mirrors ecad.Config with config:"" tags for ucfg Unpack.
	// If ecad.Config gains a new field, add it here and in the mapping
	// below; TestMinimalConfigRoundTrip (minimal_test.go) will catch any drift.
	type localConf struct {
		URL                string            `config:"ad_url" validate:"required"`
		BaseDN             string            `config:"ad_base_dn" validate:"required"`
		User               string            `config:"ad_user" validate:"required"`
		Password           string            `config:"ad_password" validate:"required"`
		Dataset            string            `config:"dataset"`
		UserQuery          string            `config:"user_query"`
		DeviceQuery        string            `config:"device_query"`
		IncludeEmptyGroups bool              `config:"include_empty_groups"`
		UserAttrs          []string          `config:"user_attributes"`
		GrpAttrs           []string          `config:"group_attributes"`
		PagingSize         uint32            `config:"ad_paging_size"`
		IDSetShards        int               `config:"idset_shards"`
		SyncInterval       time.Duration     `config:"sync_interval"`
		UpdateInterval     time.Duration     `config:"update_interval"`
		TLS                *tlscommon.Config `config:"ssl"`
	}

	d := ecad.DefaultConfig()
	lc := localConf{
		IDSetShards:    d.IDSetShards,
		SyncInterval:   d.SyncInterval,
		UpdateInterval: d.UpdateInterval,
	}
	if err := cfg.Unpack(&lc); err != nil {
		return nil, 0, 0, err
	}

	ec := ecad.Config{
		URL:                lc.URL,
		BaseDN:             lc.BaseDN,
		User:               lc.User,
		Password:           lc.Password,
		Dataset:            lc.Dataset,
		UserQuery:          lc.UserQuery,
		DeviceQuery:        lc.DeviceQuery,
		IncludeEmptyGroups: lc.IncludeEmptyGroups,
		UserAttrs:          lc.UserAttrs,
		GrpAttrs:           lc.GrpAttrs,
		PagingSize:         lc.PagingSize,
		IDSetShards:        lc.IDSetShards,
		SyncInterval:       lc.SyncInterval,
		UpdateInterval:     lc.UpdateInterval,
	}

	tc, err := buildTLS(lc.URL, lc.TLS, log)
	if err != nil {
		return nil, 0, 0, err
	}
	ec.TLS = tc

	if err := ec.Validate(); err != nil {
		return nil, 0, 0, err
	}
	p, err := ecad.New(ec)
	if err != nil {
		return nil, 0, 0, err
	}
	return p, ec.SyncInterval, ec.UpdateInterval, nil
}

func buildTLS(rawURL string, tc *tlscommon.Config, log *logp.Logger) (*tls.Config, error) {
	if tc == nil || !tc.IsEnabled() {
		return nil, nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "ldaps" {
		return nil, nil
	}

	tlsConfig, err := tlscommon.LoadTLSConfig(tc, log)
	if err != nil {
		return nil, err
	}

	host, _, err := net.SplitHostPort(u.Host)
	var addrErr *net.AddrError
	switch {
	case err == nil:
	case errors.As(err, &addrErr):
		if addrErr.Err != "missing port in address" {
			return nil, err
		}
		host = u.Host
	default:
		return nil, err
	}
	return tlsConfig.BuildModuleClientConfig(host), nil
}
