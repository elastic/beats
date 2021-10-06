// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"fmt"
	"strings"
)

type targetType string

const (
	targetBody      targetType = "body"
	targetHeader    targetType = "header"
	targetURLValue  targetType = "url.value"
	targetURLParams targetType = "url.params"
)

type errInvalidTarget struct {
	target string
}

func (err errInvalidTarget) Error() string {
	return fmt.Sprintf("invalid target: %s", err.target)
}

type targetInfo struct {
	Type targetType
	Name string
}

func getTargetInfo(t string) (targetInfo, error) {
	parts := strings.SplitN(t, ".", 2)
	if len(parts) < 2 {
		return targetInfo{}, errInvalidTarget{t}
	}
	switch parts[0] {
	case "url":
		if parts[1] == "value" {
			return targetInfo{Type: targetURLValue}, nil
		}

		paramParts := strings.SplitN(parts[1], ".", 2)
		if len(paramParts) < 2 || paramParts[0] != "params" {
			return targetInfo{}, errInvalidTarget{t}
		}

		return targetInfo{
			Type: targetURLParams,
			Name: paramParts[1],
		}, nil
	case "header":
		return targetInfo{
			Type: targetHeader,
			Name: parts[1],
		}, nil
	case "body":
		return targetInfo{
			Type: targetBody,
			Name: parts[1],
		}, nil
	}
	return targetInfo{}, errInvalidTarget{t}
}
