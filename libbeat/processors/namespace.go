// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package processors

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
)

type Namespace struct {
	reg map[string]pluginer
}

type plugin struct {
	c Constructor
}

type pluginer interface {
	Plugin() Constructor
}

func NewNamespace() *Namespace {
	return &Namespace{
		reg: map[string]pluginer{},
	}
}

func (ns *Namespace) Register(name string, factory Constructor) error {
	p := plugin{NewConditional(factory)}
	names := strings.Split(name, ".")
	if err := ns.add(names, p); err != nil {
		return fmt.Errorf("plugin %s registration fail %v", name, err)
	}
	return nil
}

func (ns *Namespace) add(names []string, p pluginer) error {
	name := names[0]

	// register plugin if intermediate node in path being processed
	if len(names) == 1 {
		if _, found := ns.reg[name]; found {
			return errors.Errorf("%v exists already", name)
		}

		ns.reg[name] = p
		return nil
	}

	// check if namespace path already exists
	tmp, found := ns.reg[name]
	if found {
		ns, ok := tmp.(*Namespace)
		if !ok {
			return errors.New("non-namespace plugin already registered")
		}
		return ns.add(names[1:], p)
	}

	// register new namespace
	sub := NewNamespace()
	err := sub.add(names[1:], p)
	if err != nil {
		return err
	}
	ns.reg[name] = sub
	return nil
}

func (ns *Namespace) Plugin() Constructor {
	return NewConditional(func(cfg *common.Config) (Processor, error) {
		var section string
		for _, name := range cfg.GetFields() {
			if name == "when" { // TODO: remove check for "when" once fields are filtered
				continue
			}

			if section != "" {
				return nil, errors.Errorf("too many lookup modules "+
					"configured (%v, %v)", section, name)
			}

			section = name
		}

		if section == "" {
			return nil, errors.New("no lookup module configured")
		}

		backend, found := ns.reg[section]
		if !found {
			return nil, errors.Errorf("unknown lookup module: %v", section)
		}

		config, err := cfg.Child(section, -1)
		if err != nil {
			return nil, err
		}

		constructor := backend.Plugin()
		return constructor(config)
	})
}

func (p plugin) Plugin() Constructor { return p.c }
