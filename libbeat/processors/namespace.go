package processors

import (
	"errors"
	"fmt"
	"strings"

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
			return errors.New("exists already")
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
	return NewConditional(func(cfg common.Config) (Processor, error) {
		var section string
		for _, name := range cfg.GetFields() {
			if name == "when" { // TODO: remove check for "when" once fields are filtered
				continue
			}

			if section != "" {
				return nil, fmt.Errorf("Too many lookup modules configured (%v, %v)",
					section, name)
			}

			section = name
		}

		if section == "" {
			return nil, errors.New("No lookup module configured")
		}

		backend, found := ns.reg[section]
		if !found {
			return nil, fmt.Errorf("Unknown lookup module: %v", section)
		}

		config, err := cfg.Child(section, -1)
		if err != nil {
			return nil, err
		}

		constructor := backend.Plugin()
		return constructor(*config)
	})
}

func (p plugin) Plugin() Constructor { return p.c }
