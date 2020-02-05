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

package processor

import (
	"github.com/dop251/goja"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/processors/script/javascript"
)

// chainBuilder builds a new processor chain.
type chainBuilder struct {
	chain
	runtime *goja.Runtime
	this    *goja.Object
}

// newChainBuilder returns a javascript constructor that constructs a
// chainBuilder.
func newChainBuilder(runtime *goja.Runtime) func(call goja.ConstructorCall) *goja.Object {
	return func(call goja.ConstructorCall) *goja.Object {
		if len(call.Arguments) > 0 {
			panic(runtime.NewGoError(errors.New("Chain accepts no arguments")))
		}

		c := &chainBuilder{runtime: runtime, this: call.This}
		for name, fn := range registry.Constructors() {
			c.this.Set(name, c.makeBuilderFunc(fn))
		}
		call.This.Set("Add", c.Add)
		call.This.Set("Build", c.Build)

		return nil
	}
}

// makeBuilderFunc returns a javascript function that constructs a new native
// beat processor and adds it to the chain.
func (b *chainBuilder) makeBuilderFunc(constructor processors.Constructor) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		return b.addProcessor(constructor, call)
	}
}

// addProcessor constructs a new native beat processor and adds it to the chain.
func (b *chainBuilder) addProcessor(constructor processors.Constructor, call goja.FunctionCall) *goja.Object {
	p, err := newNativeProcessor(constructor, call)
	if err != nil {
		panic(b.runtime.NewGoError(err))
	}

	b.procs = append(b.procs, p)
	return b.this
}

// Add adds a processor to the chain. It requires one argument that can be
// either a beat processor or a javascript function.
func (b *chainBuilder) Add(call goja.FunctionCall) goja.Value {
	a0 := call.Argument(0)
	if goja.IsUndefined(a0) {
		panic(b.runtime.NewGoError(errors.New("Add requires a processor object parameter, but got undefined")))
	}

	switch v := a0.Export().(type) {
	case *beatProcessor:
		b.procs = append(b.procs, v.p)
	case func(goja.FunctionCall) goja.Value:
		b.procs = append(b.procs, newJSProcessor(v))
	default:
		panic(b.runtime.NewGoError(errors.Errorf("arg0 must be a processor object, but got %T", a0.Export())))
	}

	return b.this
}

// Build returns a processor consisting of the previously added processors.
func (b *chainBuilder) Build(call goja.FunctionCall) goja.Value {
	if len(b.procs) == 0 {
		b.runtime.NewGoError(errors.New("no processors have been added to the chain"))
	}

	p := &beatProcessor{b.runtime, &b.chain}
	return b.runtime.ToValue(p)
}

type gojaCall interface {
	Argument(idx int) goja.Value
}

type jsFunction func(call goja.FunctionCall) goja.Value

type processor interface {
	run(event javascript.Event) error
}

// jsProcessor is a javascript function that accepts the event as a parameter.
type jsProcessor struct {
	fn   jsFunction
	call goja.FunctionCall
}

func newJSProcessor(fn jsFunction) *jsProcessor {
	return &jsProcessor{fn: fn, call: goja.FunctionCall{Arguments: make([]goja.Value, 1)}}
}

func (p *jsProcessor) run(event javascript.Event) error {
	p.call.Arguments[0] = event.JSObject()
	p.fn(p.call)
	p.call.Arguments[0] = nil
	return nil
}

// nativeProcessor is a normal Beat processor.
type nativeProcessor struct {
	processors.Processor
}

func newNativeProcessor(constructor processors.Constructor, call gojaCall) (processor, error) {
	var config *common.Config

	if a0 := call.Argument(0); !goja.IsUndefined(a0) {
		var err error
		config, err = common.NewConfigFrom(a0.Export())
		if err != nil {
			return nil, err
		}
	} else {
		// No config so use an empty config.
		config = common.NewConfig()
	}

	p, err := constructor(config)
	if err != nil {
		return nil, err
	}
	return &nativeProcessor{p}, nil
}

func (p *nativeProcessor) run(event javascript.Event) error {
	out, err := p.Processor.Run(event.Wrapped())
	if err != nil {
		return err
	}
	if out == nil {
		event.Cancel()
	}
	return nil
}

// chain is a list of processors to run serially to process an event.
type chain struct {
	procs []processor
}

func (c *chain) run(event javascript.Event) error {
	for _, p := range c.procs {
		if event.IsCancelled() {
			return nil
		}

		if err := p.run(event); err != nil {
			return err
		}
	}

	return nil
}
