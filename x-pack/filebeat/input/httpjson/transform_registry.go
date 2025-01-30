// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"fmt"
	"strings"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// registry is a collection of namespaced transform constructors.
// The registry is keyed on the namespace major and then on the
// transforms name.
type registry map[string]map[string]constructor

type constructor func(config *conf.C, log *logp.Logger) (transform, error)

var registeredTransforms = registry{
	requestNamespace: {
		appendName: newAppendRequest,
		deleteName: newDeleteRequest,
		setName:    newSetRequestPagination,
	},
	responseNamespace: {
		appendName: newAppendResponse,
		deleteName: newDeleteResponse,
		setName:    newSetResponse,
	},
	paginationNamespace: {
		appendName: newAppendPagination,
		deleteName: newDeletePagination,
		setName:    newSetRequestPagination,
	},
}

func (reg registry) get(namespace, transform string) (_ constructor, ok bool) {
	c, ok := reg[namespace][transform]
	return c, ok
}

func (reg registry) String() string {
	if len(reg) == 0 {
		return "(empty registry)"
	}

	var str string
	for namespace, m := range reg {
		names := make([]string, 0, len(m))
		for k := range m {
			names = append(names, k)
		}
		str += fmt.Sprintf("%s: (%s)\n", namespace, strings.Join(names, ", "))
	}

	return str
}
