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

package monitoring

import "strings"

type KeyValueVisitor struct {
	cb    func(key string, value interface{})
	level []string
}

func NewKeyValueVisitor(cb func(string, interface{})) *KeyValueVisitor {
	return &KeyValueVisitor{cb: cb}
}

func (vs *KeyValueVisitor) OnRegistryStart() {}

func (vs *KeyValueVisitor) OnRegistryFinished() {
	if len(vs.level) > 0 {
		vs.dropName()
	}
}

func (vs *KeyValueVisitor) OnKey(name string) {
	vs.level = append(vs.level, name)
}

func (vs *KeyValueVisitor) getName() string {
	defer vs.dropName()
	if len(vs.level) == 1 {
		return vs.level[0]
	}
	return strings.Join(vs.level, ".")
}

func (vs *KeyValueVisitor) dropName() {
	vs.level = vs.level[:len(vs.level)-1]
}

func (vs *KeyValueVisitor) OnString(s string)        { vs.cb(vs.getName(), s) }
func (vs *KeyValueVisitor) OnBool(b bool)            { vs.cb(vs.getName(), b) }
func (vs *KeyValueVisitor) OnNil()                   { vs.cb(vs.getName(), nil) }
func (vs *KeyValueVisitor) OnInt(i int64)            { vs.cb(vs.getName(), i) }
func (vs *KeyValueVisitor) OnFloat(f float64)        { vs.cb(vs.getName(), f) }
func (vs *KeyValueVisitor) OnStringSlice(f []string) { vs.cb(vs.getName(), f) }
