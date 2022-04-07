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
	"github.com/dop251/goja_nodejs/require"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/libbeat/processors"
	"github.com/elastic/beats/v8/libbeat/processors/script/javascript"
)

// Create constructors for most of the Beat processors.
// Note that script is omitted to avoid nesting.
var registry = processors.NewNamespace()

// RegisterPlugin registeres processor plugins for the javascript processor.
func RegisterPlugin(name string, c processors.Constructor) {
	logp.L().Named("javascript").Debugf("Register script processor %s", name)

	err := registry.Register(name, c)
	if err != nil {
		panic(err)
	}
}

// beatProcessor wraps a processor for javascript.
type beatProcessor struct {
	rt *goja.Runtime
	p  processor
}

func (bp *beatProcessor) Run(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) != 1 {
		panic(bp.rt.NewGoError(errors.New("Run requires one argument")))
	}

	e, ok := call.Argument(0).ToObject(bp.rt).Get("_private").Export().(javascript.Event)
	if !ok {
		panic(bp.rt.NewGoError(errors.New("arg 0 must be an Event")))
	}

	if e.IsCancelled() {
		return goja.Null()
	}

	err := bp.p.run(e)
	if err != nil {
		panic(bp.rt.NewGoError(err))
	}

	if e.IsCancelled() {
		return goja.Null()
	}

	return e.JSObject()
}

// newConstructor returns a Javascript constructor function that constructs a
// Beat processor. The constructor wraps a beat processor constructor. The
// javascript constructor must be passed a value that can be treated as the
// processor's config.
func newConstructor(
	runtime *goja.Runtime,
	constructor processors.Constructor,
) func(call goja.ConstructorCall) *goja.Object {
	return func(call goja.ConstructorCall) *goja.Object {
		p, err := newNativeProcessor(constructor, call)
		if err != nil {
			panic(runtime.NewGoError(err))
		}

		bp := &beatProcessor{runtime, p}
		return runtime.ToValue(bp).ToObject(nil)
	}
}

// Require registers the processor module that exposes constructors for beat
// processors from javascript.
//
//    // javascript
//    var processor = require('processor');
//
//    // Construct a single processor.
//    var chopLog = new processor.Dissect({tokenizer: "%{key}: %{value}"});
//
//    // Construct/compose a processor chain.
//    var mutateLog = new processor.Chain()
//        .Add(chopLog)
//        .AddProcessMetadata({match_pids: [process.pid]})
//        .Add(function(evt) {
//            evt.Put("hello", "world");
//        })
//        .Build();
//
func Require(runtime *goja.Runtime, module *goja.Object) {
	o := module.Get("exports").(*goja.Object)

	for name, fn := range registry.Constructors() {
		o.Set(name, newConstructor(runtime, fn))
	}

	// Chain returns a builder for creating a chain of processors.
	o.Set("Chain", newChainBuilder(runtime))
}

// Enable adds path to the given runtime.
func Enable(runtime *goja.Runtime) {
	runtime.Set("processor", require.Require(runtime, "processor"))
}

func init() {
	require.RegisterNativeModule("processor", Require)
}
