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

package pipeline

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCountACK(t *testing.T) {
	dummyPipeline := &Pipeline{}
	dummyFunc := func(total, acked int) {}
	cAck := newCountACK(dummyPipeline, dummyFunc)
	assert.Equal(t, dummyPipeline, cAck.pipeline)
	assert.Equal(t, reflect.ValueOf(dummyFunc).Pointer(), reflect.ValueOf(cAck.fn).Pointer())
}

func TestMakeCountACK(t *testing.T) {
	dummyPipeline := &Pipeline{}
	dummyFunc := func(total, acked int) {}
	dummySema := &sema{}
	tests := []struct {
		canDrop            bool
		sema               *sema
		fn                 func(total, acked int)
		pipeline           *Pipeline
		expectedOutputType reflect.Value
	}{
		{canDrop: false, sema: dummySema, fn: dummyFunc, pipeline: dummyPipeline, expectedOutputType: reflect.ValueOf(&countACK{})},
		{canDrop: true, sema: dummySema, fn: dummyFunc, pipeline: dummyPipeline, expectedOutputType: reflect.ValueOf(&boundGapCountACK{})},
	}
	for _, test := range tests {
		output := makeCountACK(test.pipeline, test.canDrop, test.sema, test.fn)
		assert.Equal(t, test.expectedOutputType.String(), reflect.ValueOf(output).String())
	}
}
