// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package vault

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	conf "github.com/elastic/elastic-agent-libs/config"
	ucfg "github.com/elastic/go-ucfg"

	"github.com/elastic/beats/v7/heartbeat/monitors/secretstores"
)

// referencePattern matches vault://<content>, also inside a longer value, as
// in "Bearer vault://team/app#token". The content ends at the first character
// outside its set; parseReference rejects malformed content.
var referencePattern = regexp.MustCompile(`(^|[^A-Za-z0-9+.\-])vault://([A-Za-z0-9._~%@/#\-]*)`)

// Validate checks the references in the monitor config cfg without
// contacting Vault.
func (s *store) Validate(cfg *conf.C, scope secretstores.Scope) error {
	m, _, err := referenceSettings(cfg)
	if err != nil {
		return err
	}
	return walkSettings(m, func(v string) (string, error) {
		return replaceReferences(v, scope, func(content string) (string, error) {
			_, _, err := parseReference(content)
			if err != nil {
				return "", fmt.Errorf("vault: invalid reference vault://%s: %w", content, err)
			}
			return "", nil
		})
	})
}

// Replace returns a copy of the monitor config cfg with the vault references
// replaced by their values. Without references, cfg is returned as is.
func (s *store) Replace(ctx context.Context, cfg *conf.C, scope secretstores.Scope) (*conf.C, error) {
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
		return replaceReferences(v, scope, func(content string) (string, error) {
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

// referenceSettings returns the settings of cfg to look for references in,
// with the encodedSettings holding references decoded. The process-wide
// options resolve keystore and environment references, so their values can
// be references too. If one of them doesn't resolve, resolveErr is its error
// and m keeps them as text: the plugin reports the error of a monitor
// without references.
func referenceSettings(cfg *conf.C) (m map[string]any, resolveErr error, err error) {
	if resolveErr = cfg.Unpack(&m); resolveErr == nil {
		decodeSettings(m, parseYAML)
		return m, nil, nil
	}
	m = nil
	// ResolveNOOP returns ${...} expressions as text and never fails.
	if err := (*ucfg.Config)(cfg).Unpack(&m, ucfg.PathSep("."), ucfg.ResolveNOOP); err != nil {
		return nil, nil, fmt.Errorf("could not read monitor config: %w", err)
	}
	decodeSettings(m, parseYAMLAsText)
	return m, resolveErr, nil
}

// parseYAML parses a decoded setting as the plugin does, so keystore and
// environment references in the decoded content resolve.
func parseYAML(name string, raw []byte) (map[string]any, error) {
	c, err := conf.NewConfigWithYAML(raw, name)
	if err != nil {
		return nil, err
	}
	var obj map[string]any
	return obj, c.Unpack(&obj)
}

// parseYAMLAsText parses a decoded setting keeping ${...} expressions as text.
func parseYAMLAsText(_ string, raw []byte) (map[string]any, error) {
	var obj map[string]any
	return obj, yaml.Unmarshal(raw, &obj)
}

// replaceReferences replaces the references of scope in s with the values
// returned by resolve, called with the text after "vault://" and the
// connection name, if any. References of other connections are kept.
func replaceReferences(s string, scope secretstores.Scope, resolve func(content string) (string, error)) (string, error) {
	var firstErr error
	out := referencePattern.ReplaceAllStringFunc(s, func(expr string) string {
		if firstErr != nil {
			return expr
		}
		sm := referencePattern.FindStringSubmatch(expr)
		content, mine, err := scope.Claim(sm[2])
		if err != nil {
			firstErr = fmt.Errorf("vault: invalid reference vault://%s: %w", sm[2], err)
			return expr
		}
		if !mine {
			return expr
		}
		v, err := resolve(content)
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

// encodedSettings are monitor settings that Fleet can deliver as a base64
// YAML/JSON string instead of an object, see unpackYAMLOrBase64 in the http
// monitor. References inside them are resolved after decoding.
var encodedSettings = []string{"kerberos", "ntlm"}

// decodeSettings replaces the encodedSettings of m that are base64 strings
// with references by the decoded object, which the plugins also accept.
// Settings without references are kept as they are, and so are invalid
// values, so the plugin reports the error.
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

// walkSettings calls fn on every string of the monitor settings m, except
// inline browser scripts. A secret written into script source would break on
// quotes and newlines; scripts read secrets from params instead.
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

// containsReferences reports whether the settings in v have a reference.
// Inline browser scripts are ignored, as in walkSettings.
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

// walkStrings calls fn on every string in v, a value unpacked from a config,
// and stores the result in place.
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
