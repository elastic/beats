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

package console

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/pkg/errors"

	"github.com/elastic/elastic-agent-libs/logp"

	// Require the util module for handling the log format arguments.
	_ "github.com/dop251/goja_nodejs/util"
)

// Console is a module that enables logging via the logp package (Beat logger).
type Console struct {
	runtime *goja.Runtime
	util    *goja.Object
}

func (c *Console) makeLogFunc(level logp.Level) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if format, ok := goja.AssertFunction(c.util.Get("format")); ok {
			ret, err := format(c.util, call.Arguments...)
			if err != nil {
				panic(err)
			}

			log := logp.L().Named("processor.javascript")
			switch level {
			case logp.DebugLevel:
				log.Debug(ret.String())
			case logp.InfoLevel:
				log.Info(ret.String())
			case logp.WarnLevel:
				log.Warn(ret.String())
			case logp.ErrorLevel:
				log.Error(ret.String())
			default:
				panic(errors.Errorf("unhandled logp.Level: %v", level))
			}
		} else {
			panic(c.runtime.NewTypeError("util.format is not a function"))
		}

		return nil
	}
}

// Require registers the module with the runtime.
func Require(runtime *goja.Runtime, module *goja.Object) {
	c := &Console{
		runtime: runtime,
	}

	c.util = require.Require(runtime, "util").(*goja.Object)

	o := module.Get("exports").(*goja.Object)
	o.Set("debug", c.makeLogFunc(logp.DebugLevel))
	o.Set("info", c.makeLogFunc(logp.InfoLevel))
	o.Set("log", c.makeLogFunc(logp.InfoLevel))
	o.Set("warn", c.makeLogFunc(logp.WarnLevel))
	o.Set("error", c.makeLogFunc(logp.ErrorLevel))
}

// Enable adds console to the given runtime.
func Enable(runtime *goja.Runtime) {
	runtime.Set("console", require.Require(runtime, "console"))
}

func init() {
	require.RegisterNativeModule("console", Require)
}
