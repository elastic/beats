// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:generate go run internal/gen.go > supported.go

package program

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/transpiler"
)

// ErrMissingWhen is returned when no boolean expression is defined for a program.
var ErrMissingWhen = errors.New("program must define a 'When' expression")

// Spec represents a specific program specification, it contains information about how to run the
// program and also the rules to apply to the single configuration to create a specific program
// configuration.
//
// NOTE: Current spec are build at compile time, we want to revisit that to allow other program
// to register their spec in a secure way.
type Spec struct {
	Name         string               `yaml:"name"`
	Cmd          string               `yaml:"cmd"`
	Configurable string               `yaml:"configurable"`
	Args         []string             `yaml:"args"`
	Rules        *transpiler.RuleList `yaml:"rules"`
	When         string               `yaml:"when"`
}

// ReadSpecs reads all the specs that match the provided globbing path.
func ReadSpecs(path string) ([]Spec, error) {
	var specs []Spec
	files, err := filepath.Glob(path)
	if err != nil {
		return []Spec{}, errors.New(err, "could not include spec", errors.TypeConfig)
	}

	for _, f := range files {
		b, err := ioutil.ReadFile(f)
		if err != nil {
			return []Spec{}, errors.New(err, fmt.Sprintf("could not read spec %s", f), errors.TypeConfig)
		}

		spec := Spec{}
		if err := yaml.Unmarshal(b, &spec); err != nil {
			return []Spec{}, errors.New(err, fmt.Sprintf("could not unmarshal YAML for file %s", f), errors.TypeConfig)
		}
		specs = append(specs, spec)
	}

	return specs, nil
}

// NewSpecFromBytes create a Spec from a bytes.
func NewSpecFromBytes(b []byte) (Spec, error) {
	spec := Spec{}
	if err := yaml.Unmarshal(b, &spec); err != nil {
		return Spec{}, errors.New(err, "could not unmarshal YAML", errors.TypeConfig)
	}
	return spec, nil
}

// MustReadSpecs read specs and panic on errors.
func MustReadSpecs(path string) []Spec {
	s, err := ReadSpecs(path)
	if err != nil {
		panic(err)
	}
	return s
}

// FindSpecByName find a spec by name and return it or false if we cannot find it.
func FindSpecByName(name string) (Spec, bool) {
	for _, candidate := range Supported {
		if name == candidate.Name {
			return candidate, true
		}
	}
	return Spec{}, false
}
