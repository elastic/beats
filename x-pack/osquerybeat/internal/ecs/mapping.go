// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ecs

import "strings"

const keySeparator = "."

type Doc map[string]interface{}

type MappingInfo struct {
	Field string      `json:"field,omitempty" config:"field,omitempty"`
	Value interface{} `json:"value,omitempty" config:"value,omitempty"`
}

// Mapping is ECS mapping definition where the key is the dotted ECS field name
type Mapping map[string]MappingInfo

// Map creates the copy of the values from the doc[src] key to the doc[dst] key where the dst can be nested '.' delimited key
// Source is expected to be a simple key name, the destination could be nested child node
func (m Mapping) Map(doc Doc) Doc {
	res := make(Doc)
	for dst, mi := range m {
		if mi.Value != nil {
			res.Set(dst, mi.Value)
			continue
		}
		val, ok := doc[mi.Field]
		if !ok {
			continue
		}
		res.Set(dst, val)
	}
	return res
}

func (d Doc) Get(key string) (val interface{}, ok bool) {
	keys := getKeys(key)
	node := d

	for i := 0; i < len(keys)-1; i++ {
		if keys[i] == "" {
			return nil, false
		}
		val, ok = node[keys[i]]
		if ok {
			node, ok = val.(Doc)
			if ok {
				continue
			} else {
				break
			}
		} else {
			break
		}
	}

	if node != nil {
		val, ok = node[keys[len(keys)-1]]
	}
	return
}

func (d Doc) Set(key string, val interface{}) {
	keys := getKeys(key)
	node := d

	// Create nested keys if needed
	for i := 0; i < len(keys)-1; i++ {
		if keys[i] == "" {
			return
		}

		inode, ok := node[keys[i]]
		if ok {
			node, ok = inode.(Doc)
		} else {
			d := make(Doc)
			node[keys[i]] = d
			node = d
		}
	}

	key = keys[len(keys)-1]
	if key == "" {
		return
	}
	node[key] = val
}

func getKeys(key string) []string {
	return strings.Split(key, keySeparator)
}
