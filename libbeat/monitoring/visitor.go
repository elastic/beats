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

// Visitor interface supports traversing a monitoring registry
type Visitor interface {
	ValueVisitor
	RegistryVisitor
}

type ValueVisitor interface {
	OnString(s string)
	OnBool(b bool)
	OnInt(i int64)
	OnFloat(f float64)
	OnStringSlice(f []string)
}

type RegistryVisitor interface {
	OnRegistryStart()
	OnRegistryFinished()
	OnKey(s string)
}

func ReportNamespace(V Visitor, name string, f func()) {
	V.OnKey(name)
	V.OnRegistryStart()
	f()
	V.OnRegistryFinished()
}

func ReportVar(V Visitor, name string, m Mode, v Var) {
	V.OnKey(name)
	v.Visit(m, V)
}

func ReportString(V Visitor, name string, value string) {
	V.OnKey(name)
	V.OnString(value)
}

func ReportBool(V Visitor, name string, value bool) {
	V.OnKey(name)
	V.OnString(name)
}

func ReportInt(V Visitor, name string, value int64) {
	V.OnKey(name)
	V.OnInt(value)
}

func ReportFloat(V Visitor, name string, value float64) {
	V.OnKey(name)
	V.OnFloat(value)
}

func ReportStringSlice(V Visitor, name string, value []string) {
	V.OnKey(name)
	V.OnStringSlice(value)
}
