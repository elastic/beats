// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"errors"
	"fmt"
	"strings"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
)

type constructor func(config *common.Config, log *logp.Logger) (transform, error)

var registeredTransforms = newRegistry()

type registry struct {
	namespaces map[string]map[string]constructor
}

func newRegistry() *registry {
	return &registry{namespaces: make(map[string]map[string]constructor)}
}

func (reg *registry) register(namespace, transform string, cons constructor) error {
	if cons == nil {
		return errors.New("constructor can't be nil")
	}

	m, found := reg.namespaces[namespace]
	if !found {
		reg.namespaces[namespace] = make(map[string]constructor)
		m = reg.namespaces[namespace]
	}

	if _, found := m[transform]; found {
		return errors.New("already registered")
	}

	m[transform] = cons

	return nil
}

func (reg registry) String() string {
	if len(reg.namespaces) == 0 {
		return "(empty registry)"
	}

	var str string
	for namespace, m := range reg.namespaces {
		var names []string
		for k := range m {
			names = append(names, k)
		}
		str += fmt.Sprintf("%s: (%s)\n", namespace, strings.Join(names, ", "))
	}

	return str
}

func (reg registry) get(namespace, transform string) (constructor, bool) {
	m, found := reg.namespaces[namespace]
	if !found {
		return nil, false
	}
	c, found := m[transform]
	return c, found
}

func registerTransform(namespace, transform string, constructor constructor) {
	logp.L().Named(logName).Debugf("Register transform %s:%s", namespace, transform)

	err := registeredTransforms.register(namespace, transform, constructor)
	if err != nil {
		panic(err)
	}
}
