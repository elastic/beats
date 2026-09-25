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

package testing

type nullDriver struct{}

// NullDriver does nothing, ignores all output and doesn't die on errors
var NullDriver = &nullDriver{}

func (d *nullDriver) Run(name string, f func(Driver)) {
	f(d)
}

func (d *nullDriver) Info(field, value string) {}

func (d *nullDriver) Warn(field, reason string) {}

func (d *nullDriver) Error(field string, err error) {}

func (d *nullDriver) Fatal(field string, err error) {}

func (d *nullDriver) Result(data string) {}
