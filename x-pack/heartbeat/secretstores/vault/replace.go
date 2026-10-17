// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package vault

import (
	"context"

	conf "github.com/elastic/elastic-agent-libs/config"

	"github.com/elastic/beats/v7/heartbeat/monitors/secretstores"
)

var replacer = secretstores.NewSchemeReplacer(storeType)

// Validate checks the references in the monitor config cfg without
// contacting Vault.
func (s *store) Validate(cfg *conf.C) error {
	return replacer.Validate(cfg, func(content string) error {
		_, _, err := parseReference(content)
		return err
	})
}

// Replace returns a copy of the monitor config cfg with the vault references
// replaced by their values. Without references, cfg is returned as is.
func (s *store) Replace(ctx context.Context, cfg *conf.C) (*conf.C, error) {
	return replacer.Replace(cfg, func(content string) (string, error) {
		return s.resolve(ctx, content)
	})
}

func replaceReferences(s string, resolve func(content string) (string, error)) (string, error) {
	return replacer.ReplaceString(s, resolve)
}
