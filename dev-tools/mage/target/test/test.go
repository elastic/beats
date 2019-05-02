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

package test

import "github.com/magefile/mage/mg"

var (
	testDeps []interface{}
)

// RegisterDeps registers dependencies of the Test target (register your targets
// that execute tests).
func RegisterDeps(deps ...interface{}) {
	testDeps = append(testDeps, deps...)
}

// Test runs all available tests (unitTest + integTest).
func Test() {
	mg.SerialDeps(testDeps...)
}
