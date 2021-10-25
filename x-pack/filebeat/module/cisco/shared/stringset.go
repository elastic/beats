// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build ignore
// +build ignore

package main

import (
	"sort"
	"strings"
)

type stringSet map[string]struct{}

func newStringSet(list []string) stringSet {
	r := stringSet{}
	for _, value := range list {
		if len(value) != 0 {
			r[value] = struct{}{}
		}
	}
	return r
}

func (set stringSet) merge(o stringSet) {
	for key := range o {
		set[key] = struct{}{}
	}
}

func (set stringSet) equal(other stringSet) bool {
	if len(set) != len(other) {
		return false
	}
	for k := range set {
		if _, found := other[k]; !found {
			return false
		}
	}
	return true
}

func (set stringSet) MarshalYAML() (interface{}, error) {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, nil
}

func (set stringSet) String() string {
	yaml, _ := set.MarshalYAML()
	return strings.Join(yaml.([]string), ", ")
}
