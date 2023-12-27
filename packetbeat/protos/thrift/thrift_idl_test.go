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

package thrift

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/elastic/elastic-agent-libs/logp"
)

func thriftIdlForTesting(t *testing.T, content string) *thriftIdl {
	f, _ := ioutil.TempFile("", "")
	defer os.Remove(f.Name())

	f.WriteString(content)
	f.Close()

	idl, err := newThriftIdl([]string{f.Name()})
	if err != nil {
		t.Fatal("Parsing failed:", err)
	}

	return idl
}

func TestThriftIdl_thriftReadFiles(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("thrift", "thriftdetailed"))

	idl := thriftIdlForTesting(t, `
/* simple test */
service Test {
       i32 add(1:i32 num1, 2: i32 num2)
}
`)

	methodsMap := idl.methodsByName
	if len(methodsMap) == 0 {
		t.Error("Empty methods_map")
	}
	m, exists := methodsMap["add"]
	if !exists || m.service == nil || m.method == nil ||
		m.service.Name != "Test" || m.method.Name != "add" {

		t.Error("Bad data:", m)
	}
	if *m.params[1] != "num1" || *m.params[2] != "num2" {
		t.Error("Bad params", m.params)
	}
	if len(m.exceptions) != 0 {
		t.Error("Non empty exceptions", m.exceptions)
	}
}
