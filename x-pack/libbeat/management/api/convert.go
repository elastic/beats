// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type convertFunc func(map[string]interface{}) (map[string]interface{}, error)

var mapper = map[string]convertFunc{
	".inputs":  noopConvert,
	".modules": convertMultiple,
	"output":   convertSingle,
}

var errSubTypeNotFound = errors.New("'sub_type' key not found")

var (
	subTypeKey = "sub_type"
	moduleKey  = "module"
)

func selectConverter(t string) convertFunc {
	c := noopConvert
	for k, v := range mapper {
		if strings.Index(t, k) > -1 {
			c = v
			return c
		}
	}
	return c
}

func convertSingle(m map[string]interface{}) (map[string]interface{}, error) {
	subType, err := extractSubType(m)
	if err != nil {
		return nil, err
	}

	delete(m, subTypeKey)
	newMap := map[string]interface{}{subType: m}
	return newMap, nil
}

func convertMultiple(m map[string]interface{}) (map[string]interface{}, error) {
	subType, err := extractSubType(m)
	if err != nil {
		return nil, err
	}

	v, ok := m[moduleKey]

	if ok && v != subType {
		return nil, fmt.Errorf("module key already exist in the raw document and doesn't match the 'sub_type', expecting '%s' and received '%s", subType, v)
	}

	m[moduleKey] = subType
	delete(m, subTypeKey)
	return m, nil
}

func noopConvert(m map[string]interface{}) (map[string]interface{}, error) {
	return m, nil
}

func extractSubType(m map[string]interface{}) (string, error) {
	subType, ok := m[subTypeKey]
	if !ok {
		return "", errSubTypeNotFound
	}

	k, ok := subType.(string)
	if !ok {
		return "", fmt.Errorf("invalid type for `sub_type`, expecting a string received %T", subType)
	}
	return k, nil
}
