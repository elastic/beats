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

// ValueVisitor is an interface for the monitoring visitor type
type ValueVisitor interface {
	OnString(s string)
	OnBool(b bool)
	OnInt(i int64)
	OnFloat(f float64)
	OnStringSlice(f []string)
}

// RegistryVisitor is the interface type for interacting with a monitoring registry
type RegistryVisitor interface {
	OnRegistryStart()
	OnRegistryFinished()
	OnKey(s string)
}

// ReportNamespace reports a value for a given namespace
func ReportNamespace(V Visitor, name string, f func()) {
	V.OnKey(name)
	V.OnRegistryStart()
	f()
	V.OnRegistryFinished()
}

// ReportVar reports an interface var typew for the visitor
func ReportVar(V Visitor, name string, m Mode, v Var) {
	V.OnKey(name)
	v.Visit(m, V)
}

// ReportString reports a string value for the visitor
func ReportString(V Visitor, name string, value string) {
	V.OnKey(name)
	V.OnString(value)
}

// ReportBool reports a bool for the visitor
func ReportBool(V Visitor, name string, value bool) {
	V.OnKey(name)
	V.OnBool(value)
}

// ReportInt reports an int type for the visitor
func ReportInt(V Visitor, name string, value int64) {
	V.OnKey(name)
	V.OnInt(value)
}

// ReportFloat reports a float type for the visitor
func ReportFloat(V Visitor, name string, value float64) {
	V.OnKey(name)
	V.OnFloat(value)
}

// ReportStringSlice reports a string array for the visitor
func ReportStringSlice(V Visitor, name string, value []string) {
	V.OnKey(name)
	V.OnStringSlice(value)
}
