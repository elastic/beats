// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package okta

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/entcollect"
	ecokta "github.com/elastic/entcollect/provider/okta"
)

func init() {
	err := provider.RegisterMinimalStateProvider(Name, minimalProvider)
	if err != nil {
		panic(err)
	}
}

// minimalOAuth2Conf mirrors the legacy oAuth2Config fields with config
// tags for ucfg Unpack.
type minimalOAuth2Conf struct {
	Enabled      *bool           `config:"enabled"`
	ClientID     string          `config:"client.id"`
	ClientSecret string          `config:"client.secret"`
	Scopes       []string        `config:"scopes"`
	TokenURL     string          `config:"token_url"`
	OktaJWKFile  string          `config:"jwk_file"`
	OktaJWKJSON  common.JSONBlob `config:"jwk_json"`
	OktaJWKPEM   string          `config:"jwk_pem"`
}

func (o *minimalOAuth2Conf) isEnabled() bool {
	return o != nil && (o.Enabled == nil || *o.Enabled)
}

func minimalProvider(cfg *config.C, _ *logp.Logger) (entcollect.Provider, time.Duration, time.Duration, error) {
	// localConf mirrors ecokta.Config with config:"" tags for ucfg Unpack.
	// If ecokta.Config gains a new field, add it here and in the mapping
	// below; TestMinimalConfigRoundTrip (minimal_test.go) will catch any drift.
	type localConf struct {
		OktaDomain string             `config:"okta_domain" validate:"required"`
		OktaToken  string             `config:"okta_token"`
		OAuth2     *minimalOAuth2Conf `config:"oauth2"`

		Dataset    string   `config:"dataset"`
		EnrichWith []string `config:"enrich_with"`

		BatchSize      int           `config:"batch_size"`
		SyncInterval   time.Duration `config:"sync_interval"`
		UpdateInterval time.Duration `config:"update_interval"`
		IDSetShards    int           `config:"idset_shards"`

		LimitWindow time.Duration `config:"limit_window"`
		LimitFixed  *int          `config:"limit_fixed"`
	}

	d := ecokta.DefaultConfig()
	lc := localConf{
		EnrichWith:     d.EnrichWith,
		SyncInterval:   d.SyncInterval,
		UpdateInterval: d.UpdateInterval,
		IDSetShards:    d.IDSetShards,
		LimitWindow:    d.LimitWindow,
	}
	if err := cfg.Unpack(&lc); err != nil {
		return nil, 0, 0, err
	}

	ec := ecokta.Config{
		Domain:         lc.OktaDomain,
		Token:          lc.OktaToken,
		Dataset:        lc.Dataset,
		EnrichWith:     lc.EnrichWith,
		BatchSize:      lc.BatchSize,
		SyncInterval:   lc.SyncInterval,
		UpdateInterval: lc.UpdateInterval,
		IDSetShards:    lc.IDSetShards,
		LimitWindow:    lc.LimitWindow,
		LimitFixed:     lc.LimitFixed,
	}

	if lc.OAuth2.isEnabled() {
		jwk, err := resolveJWK(lc.OAuth2)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("oauth2 jwk: %w", err)
		}
		ec.OAuth2 = &ecokta.OAuth2Config{
			ClientID:     lc.OAuth2.ClientID,
			ClientSecret: lc.OAuth2.ClientSecret,
			TokenURL:     lc.OAuth2.TokenURL,
			Scopes:       lc.OAuth2.Scopes,
			JWK:          jwk,
		}
	}

	if err := ec.Validate(); err != nil {
		return nil, 0, 0, err
	}
	return ecokta.New(ec), ec.SyncInterval, ec.UpdateInterval, nil
}

// resolveJWK converts one of the three JWK input forms (JSON, file, PEM)
// into raw JSON bytes suitable for ecokta.OAuth2Config.JWK. Returns nil
// when none are set (client-secret auth).
func resolveJWK(o *minimalOAuth2Conf) (json.RawMessage, error) {
	switch {
	case o.OktaJWKJSON != nil:
		return json.RawMessage(o.OktaJWKJSON), nil
	case o.OktaJWKFile != "":
		b, err := os.ReadFile(o.OktaJWKFile)
		if err != nil {
			return nil, fmt.Errorf("read jwk_file %q: %w", o.OktaJWKFile, err)
		}
		if !json.Valid(b) {
			return nil, fmt.Errorf("jwk_file %q does not contain valid JSON", o.OktaJWKFile)
		}
		return json.RawMessage(b), nil
	case o.OktaJWKPEM != "":
		key, err := pemPKCS8PrivateKey([]byte(o.OktaJWKPEM))
		if err != nil {
			return nil, fmt.Errorf("parse jwk_pem: %w", err)
		}
		return jose.JSONWebKey{Key: key}.MarshalJSON()
	default:
		return nil, nil
	}
}
