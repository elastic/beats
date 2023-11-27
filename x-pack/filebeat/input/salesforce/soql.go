package salesforce

import (
	"errors"
	"strings"
)

type querier struct {
	Query string
}

func (q querier) Format() (string, error) {
	if strings.TrimSpace(q.Query) == "" {
		return "", errors.New("query is empty")
	}
	return q.Query, nil
}
