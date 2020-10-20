package transforms

import (
	"errors"
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type Constructor func(config *common.Config) (Transform, error)

var registeredTransforms = newRegistry()

type registry struct {
	namespaces map[string]map[string]Constructor
}

func newRegistry() *registry {
	return &registry{namespaces: make(map[string]map[string]Constructor)}
}

func (reg *registry) register(namespace, transform string, constructor Constructor) error {
	if constructor == nil {
		return errors.New("constructor can't be nil")
	}

	m, found := reg.namespaces[namespace]
	if !found {
		reg.namespaces[namespace] = make(map[string]Constructor)
		m = reg.namespaces[namespace]
	}

	if _, found := m[transform]; found {
		return errors.New("already registered")
	}

	m[transform] = constructor

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

func (reg registry) get(namespace, transform string) (Constructor, bool) {
	m, found := reg.namespaces[namespace]
	if !found {
		return nil, false
	}
	c, found := m[transform]
	return c, found
}

func RegisterTransform(namespace, transform string, constructor Constructor) {
	logp.L().Named(logName).Debugf("Register transform %s:%s", namespace, transform)

	err := registeredTransforms.register(namespace, transform, constructor)
	if err != nil {
		panic(err)
	}
}
