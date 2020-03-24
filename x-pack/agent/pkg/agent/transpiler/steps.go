// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transpiler

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

// StepList is a container that allow the same tree to be executed on multiple defined Step.
type StepList struct {
	Steps []Step
}

type Step interface {
	Apply() error
}

// Apply applies a list of steps.
func (r *StepList) Apply() error {
	var err error
	for _, rule := range r.Steps {
		err = rule.Apply()
		if err != nil {
			return err
		}
	}

	return nil
}

// MarshalYAML marsharl a steps list to YAML.
func (r *StepList) MarshalYAML() (interface{}, error) {
	doc := make([]map[string]Step, 0, len(r.Steps))

	for _, step := range r.Steps {
		var name string
		switch step.(type) {
		case *DeleteFileStep:
			name = "delete_file"

		default:
			return nil, fmt.Errorf("unknown rule of type %T", step)
		}

		subdoc := map[string]Step{
			name: step,
		}

		doc = append(doc, subdoc)
	}
	return doc, nil
}

// UnmarshalYAML unmarshal a YAML document into a RuleList.
func (r *StepList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var unpackTo []map[string]interface{}

	err := unmarshal(&unpackTo)
	if err != nil {
		return err
	}

	// NOTE(ph): this is a bit of a hack because I want to make sure
	// the unpack strategy stay in the struct implementation and yaml
	// doesn't have a RawMessage similar to the JSON package, so partial unpack
	// is not possible.
	unpack := func(in interface{}, out interface{}) error {
		b, err := yaml.Marshal(in)
		if err != nil {
			return err
		}
		return yaml.Unmarshal(b, out)
	}

	var steps []Step

	for _, m := range unpackTo {
		ks := keys(m)
		if len(ks) > 1 {
			return fmt.Errorf("unknown rule identifier, expecting one identifier and received %d", len(ks))
		}

		name := ks[0]
		fields := m[name]

		var s Step
		switch name {
		case "delete_file":
			s = &DeleteFileStep{}
		default:
			return fmt.Errorf("unknown rule of type %s", name)
		}

		if err := unpack(fields, r); err != nil {
			return err
		}

		steps = append(steps, s)
	}
	r.Steps = steps
	return nil
}

// DeleteFileStep removes a file from disk.
type DeleteFileStep struct {
	Path string
	// FailOnMissing fails if file is already missing
	FailOnMissing bool
}

// Apply applies delete file step.
func (r *DeleteFileStep) Apply() error {
	return nil
}

// DeleteFile creates a DeleteFileStep
func DeleteFile(path string, failOnMissing bool) *DeleteFileStep {
	return &DeleteFileStep{
		Path:          path,
		FailOnMissing: failOnMissing,
	}
}
