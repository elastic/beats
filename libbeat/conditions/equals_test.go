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

package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEqualsCreate(t *testing.T) {
	config := Config{
		Equals: &Fields{fields: map[string]interface{}{
			"proc.pid": 0.08,
		}},
	}

	_, err := NewCondition(&config)
	assert.Error(t, err)
}

func TestEqualsSingleFieldPositiveMatch(t *testing.T) {
	testConfig(t, true, secdTestEvent, &Config{
		Equals: &Fields{fields: map[string]interface{}{
			"type": "process",
		}},
	})
}

func TestEqualsBooleanFieldNegativeMatch(t *testing.T) {
	testConfig(t, false, secdTestEvent, &Config{
		Equals: &Fields{fields: map[string]interface{}{
			"final": true,
		}},
	})
}

func TestEqualsMultiFieldAndTypePositiveMatch(t *testing.T) {
	testConfig(t, true, secdTestEvent, &Config{
		Equals: &Fields{fields: map[string]interface{}{
			"type":     "process",
			"proc.pid": 305,
		}},
	})
}

var (
	config = map[string]interface{}{
		"type": "process",
	}
	config1 = map[string]interface{}{
		"type":     "process",
		"proc.pid": 305,
		"final":    false,
	}
	config2 = map[string]interface{}{
		"type":             "process",
		"proc.pid":         305,
		"final":            false,
		"tags":             "error path",
		"non-existing-key": "",
	}
	config3 = map[string]interface{}{
		"type":                "process",
		"proc.pid":            305,
		"final":               false,
		"tags":                "error path",
		"non-existing-key":    "",
		"proc.cmdline":        "/usr/libexec/secd",
		"proc.cpu.start_time": 10,
	}
)

type factory func(fields map[string]interface{}) (c Condition, err error)

func benchmarkEquals(b *testing.B, f factory, fields map[string]interface{}) {
	e, err := f(fields)
	assert.NoError(b, err)
	for i := 0; i < b.N; i++ {
		e.Check(secdTestEvent)
	}
}

func equalFactory(fields map[string]interface{}) (c Condition, err error) {
	return NewEqualsCondition(fields)
}

func equal2Factory(fields map[string]interface{}) (c Condition, err error) {
	return NewEqualsCondition2(fields)
}

func equal3Factory(fields map[string]interface{}) (c Condition, err error) {
	return NewEqualsCondition3(fields)
}

func BenchmarkEqualsWith1Conditions(b *testing.B)  { benchmarkEquals(b, equalFactory, config) }
func BenchmarkEquals2With1Conditions(b *testing.B) { benchmarkEquals(b, equal2Factory, config) }
func BenchmarkEquals3With1Conditions(b *testing.B) { benchmarkEquals(b, equal3Factory, config) }

func BenchmarkEqualsWith3Conditions(b *testing.B)  { benchmarkEquals(b, equalFactory, config1) }
func BenchmarkEquals2With3Conditions(b *testing.B) { benchmarkEquals(b, equal2Factory, config1) }
func BenchmarkEquals3With3Conditions(b *testing.B) { benchmarkEquals(b, equal3Factory, config1) }

func BenchmarkEqualsWith5Conditions(b *testing.B)  { benchmarkEquals(b, equalFactory, config2) }
func BenchmarkEquals2With5Conditions(b *testing.B) { benchmarkEquals(b, equal2Factory, config2) }
func BenchmarkEquals3With5Conditions(b *testing.B) { benchmarkEquals(b, equal3Factory, config2) }

func BenchmarkEqualsWith7Conditions(b *testing.B)  { benchmarkEquals(b, equalFactory, config3) }
func BenchmarkEquals2With7Conditions(b *testing.B) { benchmarkEquals(b, equal2Factory, config3) }
func BenchmarkEquals3With7Conditions(b *testing.B) { benchmarkEquals(b, equal3Factory, config3) }
