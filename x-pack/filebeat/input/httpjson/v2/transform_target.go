package v2

import (
	"fmt"
	"strings"
)

type targetType string

const (
	targetBody      targetType = "body"
	targetCursor    targetType = "cursor"
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
	case "cursor":
		return targetInfo{
			Type: targetCursor,
			Name: parts[1],
		}, nil
	}
	return targetInfo{}, errInvalidTarget{t}
}
