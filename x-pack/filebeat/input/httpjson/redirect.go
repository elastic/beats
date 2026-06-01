// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"errors"
	"fmt"
	"net/url"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	conf "github.com/elastic/elastic-agent-libs/config"
)

var _ v2.Redirector = InputManager{}

// Redirect implements v2.Redirector. When run_as_cel is true and a
// cel.program is present, it builds a cel input config from the
// httpjson config's shared and cel-specific fields.
func (m InputManager) Redirect(cfg *conf.C) (string, *conf.C, error) {
	has, err := cfg.Has("run_as_cel", -1)
	if err != nil || !has {
		return "", nil, err
	}
	runAsCel, err := cfg.Bool("run_as_cel", -1)
	if err != nil || !runAsCel {
		return "", nil, err
	}
	has, err = cfg.Has("cel.program", -1)
	if err != nil {
		return "", nil, err
	}
	if !has {
		return "", nil, errors.New("run_as_cel requires cel.program")
	}
	newCfg, err := convertHttpjsonToCel(cfg)
	if err != nil {
		return "", nil, err
	}
	m.migrateCursor(cfg, newCfg)
	return "cel", newCfg, nil
}

// migrateCursor reads the httpjson cursor from the persistent store and
// injects it into the translated cel config's state.cursor so that the
// cel input continues from where httpjson left off. If the httpjson input
// was stateless or the store is unavailable, this is a no-op.
func (m InputManager) migrateCursor(src, dst *conf.C) {
	id, _ := src.String("id", -1) // Missing id is fine; cursorKey handles empty.
	rawURL, err := src.String("request.url", -1)
	if err != nil {
		return
	}
	// Parse and re-serialize to match source.Name() normalization.
	u, err := url.Parse(rawURL)
	if err != nil {
		return
	}
	store, err := m.cursor.StateStore.StoreFor("httpjson")
	if err != nil {
		m.cursor.Logger.Warnw("cursor migration: cannot open store", "error", err)
		return
	}
	defer store.Close()

	key := cursorKey("httpjson", id, u.String())
	var entry map[string]interface{}
	if err := store.Get(key, &entry); err != nil {
		return
	}
	cursor, ok := entry["cursor"]
	if !ok || cursor == nil {
		return
	}
	cursorCfg, err := conf.NewConfigFrom(cursor)
	if err != nil {
		m.cursor.Logger.Warnw("cursor migration: cannot create config from cursor", "error", err)
		return
	}

	// Ensure the state sub-config exists before injecting.
	has, err := dst.Has("state", -1)
	if err != nil {
		return
	}
	if !has {
		if err := dst.SetChild("state", -1, conf.NewConfig()); err != nil {
			return
		}
	}
	if err := dst.SetChild("state.cursor", -1, cursorCfg); err != nil {
		m.cursor.Logger.Warnw("cursor migration: cannot inject cursor into config", "error", err)
	}
}

func cursorKey(typ, id, url string) string {
	if id != "" {
		return fmt.Sprintf("%s::%s::%s", typ, id, url)
	}
	return fmt.Sprintf("%s::%s", typ, url)
}

// convertHttpjsonToCel builds a cel input config from an httpjson config
// by extracting shared fields and cel-namespaced fields.
func convertHttpjsonToCel(cfg *conf.C) (*conf.C, error) {
	out := conf.NewConfig()

	if err := out.SetString("type", -1, "cel"); err != nil {
		return nil, fmt.Errorf("cannot set type: %w", err)
	}

	// Copy shared string fields that map directly or with a rename.
	// Durations and URLs are stored as strings in the config.
	scalars := []struct{ src, dst string }{
		{"interval", "interval"},
		{"request.url", "resource.url"},
		{"request.timeout", "resource.timeout"},
		{"request.proxy_url", "resource.proxy_url"},
		{"request.idle_connection_timeout", "resource.idle_connection_timeout"},
	}
	for _, f := range scalars {
		has, err := cfg.Has(f.src, -1)
		if err != nil {
			return nil, fmt.Errorf("checking %q: %w", f.src, err)
		}
		if !has {
			continue
		}
		v, err := cfg.String(f.src, -1)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", f.src, err)
		}
		if err := out.SetString(f.dst, -1, v); err != nil {
			return nil, fmt.Errorf("setting %q: %w", f.dst, err)
		}
	}

	// Copy sub-configs that transfer as a block.
	blocks := []struct{ src, dst string }{
		{"auth", "auth"},
		{"request.retry", "resource.retry"},
		{"request.redirect", "resource.redirect"},
		{"request.keep_alive", "resource.keep_alive"},
		{"request.tracer", "resource.tracer"},
		{"request.ssl", "resource.ssl"},
		{"request.proxy_headers", "resource.proxy_headers"},
	}
	for _, b := range blocks {
		has, err := cfg.Has(b.src, -1)
		if err != nil {
			return nil, fmt.Errorf("checking %q: %w", b.src, err)
		}
		if !has {
			continue
		}
		sub, err := cfg.Child(b.src, -1)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", b.src, err)
		}
		if err := out.SetChild(b.dst, -1, sub); err != nil {
			return nil, fmt.Errorf("setting %q: %w", b.dst, err)
		}
	}

	// Copy shared boolean fields.
	bools := []struct{ src, dst string }{
		{"request.proxy_disable", "resource.proxy_disable"},
	}
	for _, f := range bools {
		has, err := cfg.Has(f.src, -1)
		if err != nil {
			return nil, fmt.Errorf("checking %q: %w", f.src, err)
		}
		if !has {
			continue
		}
		v, err := cfg.Bool(f.src, -1)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", f.src, err)
		}
		if err := out.SetBool(f.dst, -1, v); err != nil {
			return nil, fmt.Errorf("setting %q: %w", f.dst, err)
		}
	}

	// Extract cel-namespaced string fields into their top-level equivalents.
	celStrings := []struct{ src, dst string }{
		{"cel.program", "program"},
	}
	for _, f := range celStrings {
		has, err := cfg.Has(f.src, -1)
		if err != nil {
			return nil, fmt.Errorf("checking %q: %w", f.src, err)
		}
		if !has {
			continue
		}
		v, err := cfg.String(f.src, -1)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", f.src, err)
		}
		if err := out.SetString(f.dst, -1, v); err != nil {
			return nil, fmt.Errorf("setting %q: %w", f.dst, err)
		}
	}

	// Extract cel-namespaced integer fields.
	celInts := []struct{ src, dst string }{
		{"cel.max_executions", "max_executions"},
	}
	for _, f := range celInts {
		has, err := cfg.Has(f.src, -1)
		if err != nil {
			return nil, fmt.Errorf("checking %q: %w", f.src, err)
		}
		if !has {
			continue
		}
		v, err := cfg.Int(f.src, -1)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", f.src, err)
		}
		if err := out.SetInt(f.dst, -1, v); err != nil {
			return nil, fmt.Errorf("setting %q: %w", f.dst, err)
		}
	}

	// Extract cel sub-configs.
	celBlocks := []struct{ src, dst string }{
		{"cel.state", "state"},
		{"cel.regexp", "regexp"},
		{"cel.xsd", "xsd"},
		{"cel.redact", "redact"},
	}
	for _, b := range celBlocks {
		has, err := cfg.Has(b.src, -1)
		if err != nil {
			return nil, fmt.Errorf("checking %q: %w", b.src, err)
		}
		if !has {
			continue
		}
		sub, err := cfg.Child(b.src, -1)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", b.src, err)
		}
		if err := out.SetChild(b.dst, -1, sub); err != nil {
			return nil, fmt.Errorf("setting %q: %w", b.dst, err)
		}
	}

	// Copy passthrough fields.
	passthrough := []string{"id"}
	for _, key := range passthrough {
		has, err := cfg.Has(key, -1)
		if err != nil {
			return nil, fmt.Errorf("checking %q: %w", key, err)
		}
		if !has {
			continue
		}
		v, err := cfg.String(key, -1)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", key, err)
		}
		if err := out.SetString(key, -1, v); err != nil {
			return nil, fmt.Errorf("setting %q: %w", key, err)
		}
	}

	return out, nil
}
