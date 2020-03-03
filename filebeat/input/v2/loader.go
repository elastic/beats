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

package v2

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"

	"github.com/elastic/go-concert/ctxtool"
	"golang.org/x/sync/errgroup"
)

// Loader is used to create an Input instance. The input created
// only represents a configured object. The input MUST NOT start any
// processing yet.
type Loader interface {
	Configure(*common.Config) (Input, error)
}

// LoaderList combines multiple Loaders into one Loader.
type LoaderList []Loader

// LoaderTable combines multiple Loaders, prefixing each loader with a custom name. This allows
// multiple loaders to have similar plugin types, but still have unique names.
// When loading the type name is split by `<prefix>/<name>`. The `<prefix>` is
// used for table lookup. The rest will be passed to the found loader.
// If the 'type' name does not have a prefix, the loader table will use the
// loader registered with the empty string.
type LoaderTable struct {
	typeField string
	table     map[string]Loader
}

// ConfigsLoader uses a transformer to generate the final configuration to be passed to a loader.
// The transformer can generate multiple configurations. An input for each configuration will be generated.
// The generated inputs will be combined into one common input that runs the given inputs concurrently.
type ConfigsLoader struct {
	// The field that will be used to select the type from the configuration.
	TypeField string

	// Transform transforms the configuration into a set of input configurations.
	// Each configuration will be passed to the given loader.
	Transform ConfigTransformer
	Loader    Loader

	// Strict requires the input type to be always passed to the transformer. If set ConfigsLoader will fail
	// if the type name is unknown to the Transformer. If Strict is set to false, then the configuration is passed
	// directly to the loader if the type name is unknown to the transformer.
	Strict bool

	// Rescursive, if set instructs the ConfigsLoader to feed generated configuration back to the Transformer.
	// This allows transformations to reference other transformation inputs. THe ConfigsLoader keeps track of used
	// names and will fail if it detects a loop.
	Recusrive bool
}

// ConfigTransformer creates multiple input configurations based on a given input configuration.
type ConfigTransformer interface {
	Has(name string) bool
	Transform(cfg *common.Config) ([]*common.Config, error)
}

var _ Loader = (LoaderList)(nil)
var _ Loader = (*LoaderTable)(nil)
var _ Loader = (*ConfigsLoader)(nil)

// Add adds another loader to the list.
// Warning: a loader list must not be added to itself (or cross references between ListLoaders),
//          so to not run into an infinite loop when trying to create an input.
func (l *LoaderList) Add(other Loader) {
	*l = append(*l, other)
}

// Configure asks each loader to create an input. The first loader that creates an
// input without an error wins. If an input with configuration error is
// returned we will hold on to it, reporting the last configuration
// error we have witnessed.
func (l LoaderList) Configure(cfg *common.Config) (Input, error) {
	var lastErr error
	var lastInput Input

	for _, loader := range l {
		input, err := loader.Configure(cfg)
		if input.Run != nil {
			if err == nil {
				return input, nil
			}

			lastInput = input
			lastErr = err
		} else if lastInput.Run == nil {
			lastErr = mergeLoadError(lastErr, err)
		}
	}

	return lastInput, lastErr
}

// NewLoaderTable creates a new LoaderTable, that will select loaders based on the typeField.
func NewLoaderTable(typeField string, optLoaders map[string]Loader) *LoaderTable {
	t := &LoaderTable{
		typeField: typeField,
		table:     make(map[string]Loader, len(optLoaders)),
	}
	for name, l := range optLoaders {
		t.table[name] = l
	}
	return t
}

// Add adds another loader to the table. The default loader can be set by passing an empty string.
// Add returns an error if the name is already in use.
func (t *LoaderTable) Add(name string, l Loader) error {
	if _, exists := t.table[name]; exists {
		return fmt.Errorf("loader name '%v' already registered", l)
	}

	t.table[name] = l
	return nil
}

// Configure loads an input by looking up the loader in the table. The 'type'
// name is split by '/'.  If the type has no '/', then we fallback to the
// loader registered with "".
func (t *LoaderTable) Configure(cfg *common.Config) (Input, error) {
	fullName, err := getTypeName(cfg, t.typeField)
	if err != nil {
		return Input{}, err
	}

	var key string
	typeName := fullName
	idx := strings.IndexRune(typeName, '/')
	if idx >= 0 {
		key = typeName[:idx]
		typeName = typeName[idx+1:]
	}

	loader := t.table[key]
	if loader == nil {
		return Input{}, &LoaderError{
			Name:    fullName,
			Reason:  ErrUnknown,
			Message: fmt.Sprintf("no plugin namespace for '%v' defined", key),
		}
	}

	input, err := loader.Configure(cfg)
	if err != nil {
		if lerr, ok := err.(*LoaderError); ok {
			lerr.Name = fullName
		}
	}
	return input, err
}

// Configure configures the inputs that are generated by the transformer based
// on the given configuration.
func (cl *ConfigsLoader) Configure(cfg *common.Config) (Input, error) {
	if cl.Transform == nil || cl.Loader == nil {
		panic("invalid configs loader")
	}

	fieldName := cl.TypeField
	if fieldName == "" {
		fieldName = "type"
	}

	inputs, err := cl.load(fieldName, nil, cfg)
	if err != nil {
		return Input{}, err
	}

	inputName, _ := getTypeName(cfg, fieldName)
	if len(inputs) == 1 {
		input := inputs[0]
		input.Name = inputName
		return input, nil
	}

	return Input{
		Name: inputName,
		Run: func(ctx Context, conn beat.PipelineConnector) error {
			grp, grpContext := errgroup.WithContext(ctxtool.FromCanceller(ctx.Cancelation))
			ctx.Cancelation = grpContext
			for _, input := range inputs {
				grp.Go(func() error {
					return input.Run(ctx, conn)
				})
			}
			return grp.Wait()
		},
		Test: func(ctx TestContext) error {
			grp, grpContext := errgroup.WithContext(ctxtool.FromCanceller(ctx.Cancelation))
			ctx.Cancelation = grpContext
			for _, input := range inputs {
				if input.Test == nil {
					continue
				}
				grp.Go(func() error {
					return input.Test(ctx)
				})
			}
			return grp.Wait()
		},
	}, nil
}

func (cl *ConfigsLoader) load(
	fieldName string,
	visited []string,
	cfg *common.Config,
) ([]Input, error) {
	inputName, err := getTypeName(cfg, fieldName)
	if err != nil {
		return nil, err
	}

	// load root configuration
	if len(visited) == 0 {
		has := cl.Transform.Has(inputName)
		if !has {
			if cl.Strict {
				return nil, &LoaderError{Name: inputName, Reason: ErrUnknown}
			}
			return cl.loadInput(cfg)
		}
		return cl.loadChildren(fieldName, visited, inputName, cfg)
	}

	if !cl.Recusrive {
		return cl.loadInput(cfg)
	}

	// recursion enabled, let's give it a try...
	if alreadySeen(visited, inputName) {
		return nil, &LoaderError{Name: inputName, Reason: ErrInfiniteLoadLoop}
	}
	if !cl.Transform.Has(inputName) {
		return cl.loadInput(cfg)
	}
	return cl.loadChildren(fieldName, visited, inputName, cfg)
}

func (cl *ConfigsLoader) loadInput(cfg *common.Config) ([]Input, error) {
	input, err := cl.Loader.Configure(cfg)
	if err != nil {
		return nil, err
	}
	return []Input{input}, err
}

func (cl *ConfigsLoader) loadChildren(
	fieldName string,
	visited []string,
	name string,
	cfg *common.Config,
) ([]Input, error) {
	cfgs, err := cl.Transform.Transform(cfg)
	if err != nil {
		return nil, err
	}

	var inputs []Input
	visited = append(visited, name)
	for _, childCfg := range cfgs {
		childInputs, err := cl.load(fieldName, visited, childCfg)
		if err != nil {
			return nil, &LoaderError{
				Name:    name,
				Reason:  err,
				Message: "failed to load child inputs",
			}
		}

		inputs = append(inputs, childInputs...)
	}
	return inputs, nil
}

func alreadySeen(visited []string, name string) bool {
	for _, seen := range visited {
		if seen == name {
			return true
		}
	}
	return false
}

func mergeLoadError(err1, err2 error) error {
	if failedInputName(err1) != "" && failedInputName(err2) == "" {
		return err1
	}
	return err2
}

func getTypeName(cfg *common.Config, field string) (string, error) {
	return cfg.String(field, -1)
}
