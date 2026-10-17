// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cyberark

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	conf "github.com/elastic/elastic-agent-libs/config"
	ucfg "github.com/elastic/go-ucfg"
)

// referencePattern matches cyberark://<content>, also inside a longer value,
// as in "Bearer cyberark://Safe/App#password". The content ends at the first
// character outside its set; parseReference rejects malformed content.
var referencePattern = regexp.MustCompile(`(^|[^A-Za-z0-9+.\-])cyberark://([A-Za-z0-9._~%@/#\-]*)`)

// Validate checks the references in the monitor config cfg without
// contacting CyberArk.
func (s *store) Validate(cfg *conf.C) error {
	m, _, err := referenceSettings(cfg)
	if err != nil {
		return err
	}
	return walkSettings(m, func(v string) (string, error) {
		return replaceReferences(v, func(content string) (string, error) {
			_, _, err := parseReference(content)
			if err != nil {
				return "", fmt.Errorf("cyberark: invalid reference cyberark://%s: %w", content, err)
			}
			return "", nil
		})
	})
}

// Replace returns a copy of the monitor config cfg with the cyberark
// references replaced by their values. Without references, cfg is returned as is.
func (s *store) Replace(ctx context.Context, cfg *conf.C) (*conf.C, error) {
	m, resolveErr, err := referenceSettings(cfg)
	if err != nil {
		return nil, err
	}
	if !containsReferences(m) {
		return cfg, nil
	}
	if resolveErr != nil {
		return nil, fmt.Errorf("could not read monitor config: %w", resolveErr)
	}

	err = walkSettings(m, func(v string) (string, error) {
		return replaceReferences(v, func(content string) (string, error) {
			return s.resolve(ctx, content)
		})
	})
	if err != nil {
		return nil, err
	}

	// No VarExp: resolved values must stay literal. PathSep("") keeps keys
	// such as dotted browser params literal; the map is already nested.
	resolved, err := ucfg.NewFrom(m, ucfg.PathSep(""))
	if err != nil {
		return nil, fmt.Errorf("could not build resolved config: %w", err)
	}
	return (*conf.C)(resolved), nil
}

func referenceSettings(cfg *conf.C) (m map[string]any, resolveErr error, err error) {
	if resolveErr = cfg.Unpack(&m); resolveErr == nil {
		decodeSettings(m, parseYAML)
		return m, nil, nil
	}
	m = nil
	if err := (*ucfg.Config)(cfg).Unpack(&m, ucfg.PathSep("."), ucfg.ResolveNOOP); err != nil {
		return nil, nil, fmt.Errorf("could not read monitor config: %w", err)
	}
	decodeSettings(m, parseYAMLAsText)
	return m, resolveErr, nil
}

func parseYAML(name string, raw []byte) (map[string]any, error) {
	c, err := conf.NewConfigWithYAML(raw, name)
	if err != nil {
		return nil, err
	}
	var obj map[string]any
	return obj, c.Unpack(&obj)
}

func parseYAMLAsText(_ string, raw []byte) (map[string]any, error) {
	var obj map[string]any
	return obj, yaml.Unmarshal(raw, &obj)
}

func replaceReferences(s string, resolve func(content string) (string, error)) (string, error) {
	var firstErr error
	out := referencePattern.ReplaceAllStringFunc(s, func(expr string) string {
		if firstErr != nil {
			return expr
		}
		sm := referencePattern.FindStringSubmatch(expr)
		v, err := resolve(sm[2])
		if err != nil {
			firstErr = err
			return expr
		}
		return sm[1] + v
	})
	if firstErr != nil {
		return "", firstErr
	}
	return out, nil
}

var encodedSettings = []string{"kerberos", "ntlm"}

func decodeSettings(m map[string]any, parse func(name string, raw []byte) (map[string]any, error)) {
	for _, name := range encodedSettings {
		s, ok := m[name].(string)
		if !ok || strings.TrimSpace(s) == "" {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
		if err != nil {
			continue
		}
		obj, err := parse(name, raw)
		if err != nil || !containsReferences(obj) {
			continue
		}
		m[name] = obj
	}
}

func walkSettings(m map[string]any, fn func(string) (string, error)) error {
	inline, _ := nestedMap(m, "source", "inline")
	script, hasScript := inline["script"]
	if hasScript {
		delete(inline, "script")
		defer func() { inline["script"] = script }()
	}
	_, err := walkStrings(m, fn)
	return err
}

func nestedMap(m map[string]any, keys ...string) (map[string]any, bool) {
	for _, k := range keys {
		next, ok := m[k].(map[string]any)
		if !ok {
			return nil, false
		}
		m = next
	}
	return m, true
}

func containsReferences(v any) bool {
	var found bool
	visit := func(s string) (string, error) {
		found = found || referencePattern.MatchString(s)
		return s, nil
	}
	if m, ok := v.(map[string]any); ok {
		_ = walkSettings(m, visit)
	} else {
		_, _ = walkStrings(v, visit)
	}
	return found
}

func walkStrings(v any, fn func(string) (string, error)) (any, error) {
	switch val := v.(type) {
	case string:
		return fn(val)
	case map[string]any:
		for k, item := range val {
			res, err := walkStrings(item, fn)
			if err != nil {
				return nil, err
			}
			val[k] = res
		}
		return val, nil
	case []any:
		for i, item := range val {
			res, err := walkStrings(item, fn)
			if err != nil {
				return nil, err
			}
			val[i] = res
		}
		return val, nil
	default:
		return v, nil
	}
}
