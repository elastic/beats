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

package main

import (
	"fmt"
	"testing"
)

func Test_LineWithoutComments(t *testing.T) {
	result := lineWithoutComments("hello # world")
	if result != "hello" {
		t.Fatal(result)
	}
}

func Test_VariablesFromLine(t *testing.T) {
	l := "asfasdfs asfsfas ${HELLO}/${WORLD} sdfafsdf ${ELASTIC} ${HELLO} $(strip $(GOX_OS))"
	results := make(map[string]*struct{})

	variablesFromLine(l, results)
	fmt.Printf("%#v\n", results)

	if results["HELLO"] == nil {
		t.Fail()
	}

	if results["WORLD"] == nil {
		t.Fail()
	}

	if results["ELASTIC"] == nil {
		t.Fail()
	}

	if results["GOX_OS"] == nil {
		t.Fail()
	}
}
