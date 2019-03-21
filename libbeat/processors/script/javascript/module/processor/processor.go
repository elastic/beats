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

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/processors/actions"
	"github.com/elastic/beats/libbeat/processors/add_cloud_metadata"
	"github.com/elastic/beats/libbeat/processors/add_docker_metadata"
	"github.com/elastic/beats/libbeat/processors/add_host_metadata"
	"github.com/elastic/beats/libbeat/processors/add_kubernetes_metadata"
	"github.com/elastic/beats/libbeat/processors/add_locale"
	"github.com/elastic/beats/libbeat/processors/add_process_metadata"
	"github.com/elastic/beats/libbeat/processors/communityid"
	"github.com/elastic/beats/libbeat/processors/dissect"
	"github.com/elastic/beats/libbeat/processors/dns"
	"github.com/elastic/beats/libbeat/processors/script/javascript"
)

// newConstructor returns a JS constructor function. The constructor wraps a
// beat processor constructor. The javascript constructor must be passed a value
// that can be treated as the processor's config.
func newConstructor(
	s *goja.Runtime,
	constructor processors.Constructor,
) func(call goja.ConstructorCall) *goja.Object {
	return func(call goja.ConstructorCall) *goja.Object {
		a0 := call.Argument(0)
		if a0 == nil {
			panic(s.ToValue("constructor requires a configuration arg"))
		}

		commonConfig, err := common.NewConfigFrom(a0.Export())
		if err != nil {
			panic(s.NewGoError(err))
		}

		p, err := constructor(commonConfig)
		if err != nil {
			panic(s.NewGoError(err))
		}

		bp := &beatProcessor{s, p}
		call.This.Set("Run", bp.Run)
		return nil
	}
}

// beatProcessor wraps a beat processor for javascript.
type beatProcessor struct {
	runtime *goja.Runtime
	p       processors.Processor
}

func (bp *beatProcessor) Run(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) != 1 {
		panic(bp.runtime.NewGoError(errors.New("Run requires one argument")))
	}

	e, ok := call.Argument(0).ToObject(bp.runtime).Get("_private").Export().(javascript.Event)
	if !ok {
		panic(bp.runtime.NewGoError(errors.New("arg 0 must be an Event")))
	}

	if e.IsCancelled() {
		return goja.Null()
	}

	beatEvent, err := bp.p.Run(e.Wrapped())
	if err != nil {
		panic(bp.runtime.NewGoError(err))
	}

	if beatEvent == nil {
		e.Cancel()
		return goja.Null()
	}

	return e.JSObject()
}

// Require registers the processor module that exposes constructors for beat
// processors from javascript.
//
//    // javascript
//    var processor = require('processor');
//    var chopLog = new processor.Dissect({tokenizer: "%{key}: %{value}"});
//
func Require(runtime *goja.Runtime, module *goja.Object) {
	o := module.Get("exports").(*goja.Object)

	// Create constructors for most of the Beat processors.
	// Note that script to avoid nesting. And some of the actions like rename
	// and add_tags are omitted because those can be done natively in JS.
	o.Set("AddCloudMetadata", newConstructor(runtime, add_cloud_metadata.New))
	o.Set("AddDockerMetadata", newConstructor(runtime, add_docker_metadata.New))
	o.Set("AddHostMetadata", newConstructor(runtime, add_host_metadata.New))
	o.Set("AddKubernetesMetadata", newConstructor(runtime, add_kubernetes_metadata.New))
	o.Set("AddLocale", newConstructor(runtime, add_locale.New))
	o.Set("AddProcessMetadata", newConstructor(runtime, add_process_metadata.New))
	o.Set("CommunityID", newConstructor(runtime, communityid.New))
	o.Set("DecodeJSONFields", newConstructor(runtime, actions.NewDecodeJSONFields))
	o.Set("Dissect", newConstructor(runtime, dissect.NewProcessor))
	o.Set("DNS", newConstructor(runtime, dns.New))
}

// Enable adds path to the given runtime.
func Enable(runtime *goja.Runtime) {
	runtime.Set("processor", require.Require(runtime, "processor"))
}

func init() {
	require.RegisterNativeModule("processor", Require)
}
