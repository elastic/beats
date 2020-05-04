// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonschema

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
)

type resource struct {
	url     string
	doc     interface{}
	draft   *Draft
	schemas map[string]*Schema
}

// DecodeJSON decodes json document from r.
//
// Note that number is decoded into json.Number instead of as a float64
func DecodeJSON(r io.Reader) (interface{}, error) {
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	var doc interface{}
	if err := decoder.Decode(&doc); err != nil {
		return nil, err
	}
	if t, _ := decoder.Token(); t != nil {
		return nil, fmt.Errorf("invalid character %v after top-level value", t)
	}
	return doc, nil
}

func newResource(base string, r io.Reader) (*resource, error) {
	if strings.IndexByte(base, '#') != -1 {
		panic(fmt.Sprintf("BUG: newResource(%q)", base))
	}
	doc, err := DecodeJSON(r)
	if err != nil {
		return nil, fmt.Errorf("parsing %q failed. Reason: %v", base, err)
	}
	return &resource{
		url:     base,
		doc:     doc,
		schemas: make(map[string]*Schema)}, nil
}

func resolveURL(base, ref string) (string, error) {
	if ref == "" {
		return base, nil
	}

	refURL, err := url.Parse(ref)
	if err != nil {
		return "", err
	}
	if refURL.IsAbs() {
		return normalize(ref), nil
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	if baseURL.IsAbs() {
		return normalize(baseURL.ResolveReference(refURL).String()), nil
	}

	// filepath resolving
	base, _ = split(base)
	ref, fragment := split(ref)
	if ref == "" {
		return base + fragment, nil
	}
	dir, _ := filepath.Split(base)
	return filepath.Join(dir, ref) + fragment, nil
}

func (r *resource) resolvePtr(ptr string) (string, interface{}, error) {
	if !strings.HasPrefix(ptr, "#/") {
		panic(fmt.Sprintf("BUG: resolvePtr(%q)", ptr))
	}
	base := r.url
	p := strings.TrimPrefix(ptr, "#/")
	doc := r.doc
	for _, item := range strings.Split(p, "/") {
		item = strings.Replace(item, "~1", "/", -1)
		item = strings.Replace(item, "~0", "~", -1)
		item, err := url.PathUnescape(item)
		if err != nil {
			return "", nil, errors.New("unable to url unscape: " + item)
		}
		switch d := doc.(type) {
		case map[string]interface{}:
			if id, ok := d[r.draft.id]; ok {
				if id, ok := id.(string); ok {
					if base, err = resolveURL(base, id); err != nil {
						return "", nil, err
					}
				}
			}
			doc = d[item]
		case []interface{}:
			index, err := strconv.Atoi(item)
			if err != nil {
				return "", nil, fmt.Errorf("invalid $ref %q, reason: %s", ptr, err)
			}
			if index < 0 || index >= len(d) {
				return "", nil, fmt.Errorf("invalid $ref %q, reason: array index outofrange", ptr)
			}
			doc = d[index]
		default:
			return "", nil, errors.New("invalid $ref " + ptr)
		}
	}
	return base, doc, nil
}

func split(uri string) (string, string) {
	hash := strings.IndexByte(uri, '#')
	if hash == -1 {
		return uri, "#"
	}
	return uri[0:hash], uri[hash:]
}

func normalize(url string) string {
	base, fragment := split(url)
	if rootFragment(fragment) {
		fragment = "#"
	}
	return base + fragment
}

func rootFragment(fragment string) bool {
	return fragment == "" || fragment == "#" || fragment == "#/"
}

func resolveIDs(draft *Draft, base string, v interface{}, ids map[string]map[string]interface{}) error {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	if id, ok := m[draft.id]; ok {
		b, err := resolveURL(base, id.(string))
		if err != nil {
			return err
		}
		base = b
		ids[base] = m
	}

	for _, pname := range []string{"not", "additionalProperties"} {
		if m, ok := m[pname]; ok {
			if err := resolveIDs(draft, base, m, ids); err != nil {
				return err
			}
		}
	}

	for _, pname := range []string{"allOf", "anyOf", "oneOf"} {
		if arr, ok := m[pname]; ok {
			for _, m := range arr.([]interface{}) {
				if err := resolveIDs(draft, base, m, ids); err != nil {
					return err
				}
			}
		}
	}

	for _, pname := range []string{"definitions", "properties", "patternProperties", "dependencies"} {
		if props, ok := m[pname]; ok {
			for _, m := range props.(map[string]interface{}) {
				if err := resolveIDs(draft, base, m, ids); err != nil {
					return err
				}
			}
		}
	}

	if items, ok := m["items"]; ok {
		switch items := items.(type) {
		case map[string]interface{}:
			if err := resolveIDs(draft, base, items, ids); err != nil {
				return err
			}
		case []interface{}:
			for _, item := range items {
				if err := resolveIDs(draft, base, item, ids); err != nil {
					return err
				}
			}
		}
		if additionalItems, ok := m["additionalItems"]; ok {
			if additionalItems, ok := additionalItems.(map[string]interface{}); ok {
				if err := resolveIDs(draft, base, additionalItems, ids); err != nil {
					return err
				}
			}
		}
	}

	if draft.version >= 6 {
		for _, pname := range []string{"propertyNames", "contains"} {
			if m, ok := m[pname]; ok {
				if err := resolveIDs(draft, base, m, ids); err != nil {
					return err
				}
			}
		}
	}

	if draft.version >= 7 {
		if iff, ok := m["if"]; ok {
			if err := resolveIDs(draft, base, iff, ids); err != nil {
				return err
			}
			for _, pname := range []string{"then", "else"} {
				if m, ok := m[pname]; ok {
					if err := resolveIDs(draft, base, m, ids); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}
