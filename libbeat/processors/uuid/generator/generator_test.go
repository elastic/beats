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

package generator

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/elastic/beats/libbeat/processors/uuid/generator/elasticsearch"

	"github.com/stretchr/testify/assert"
)

func TestFactory(t *testing.T) {
	tests := map[string]struct {
		expectedGeneratorFn Fn
		expectedErr         error
	}{
		"elasticsearch": {
			elasticsearch.GetBase64UUID,
			nil,
		},
		"foobar": {
			nil,
			makeErrUnknownType("foobar"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			typ := name
			fn, err := Factory(typ)
			if test.expectedGeneratorFn != nil {
				fnName := getGeneratorFuncName(fn)
				expectedFnName := getGeneratorFuncName(test.expectedGeneratorFn)
				assert.Equal(t, fnName, expectedFnName)
			}
			if test.expectedErr != nil {
				assert.EqualError(t, err, test.expectedErr.Error())
			}
		})
	}
}

func getGeneratorFuncName(fn Fn) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}
