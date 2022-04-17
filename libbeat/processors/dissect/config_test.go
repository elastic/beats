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

package dissect

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/common"
)

func TestConfig(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		c, err := common.NewConfigFrom(map[string]interface{}{
			"tokenizer": "%{value1}",
			"field":     "message",
		})
		if !assert.NoError(t, err) {
			return
		}

		cfg := config{}
		err = c.Unpack(&cfg)
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, trimModeNone, cfg.TrimValues)
	})

	t.Run("invalid", func(t *testing.T) {
		c, err := common.NewConfigFrom(map[string]interface{}{
			"tokenizer": "%value1}",
			"field":     "message",
		})
		if !assert.NoError(t, err) {
			return
		}

		cfg := config{}
		err = c.Unpack(&cfg)
		if !assert.Error(t, err) {
			return
		}
	})

	t.Run("with tokenizer missing", func(t *testing.T) {
		c, err := common.NewConfigFrom(map[string]interface{}{})
		if !assert.NoError(t, err) {
			return
		}

		cfg := config{}
		err = c.Unpack(&cfg)
		if !assert.Error(t, err) {
			return
		}
	})

	t.Run("with empty tokenizer", func(t *testing.T) {
		c, err := common.NewConfigFrom(map[string]interface{}{
			"tokenizer": "",
		})
		if !assert.NoError(t, err) {
			return
		}

		cfg := config{}
		err = c.Unpack(&cfg)
		if !assert.Error(t, err) {
			return
		}
	})

	t.Run("tokenizer with no field defined", func(t *testing.T) {
		c, err := common.NewConfigFrom(map[string]interface{}{
			"tokenizer": "hello world",
		})
		if !assert.NoError(t, err) {
			return
		}

		cfg := config{}
		err = c.Unpack(&cfg)
		if !assert.Error(t, err) {
			return
		}
	})

	t.Run("with wrong trim_mode", func(t *testing.T) {
		c, err := common.NewConfigFrom(map[string]interface{}{
			"tokenizer":   "hello %{what}",
			"field":       "message",
			"trim_values": "bananas",
		})
		if !assert.NoError(t, err) {
			return
		}

		cfg := config{}
		err = c.Unpack(&cfg)
		if !assert.Error(t, err) {
			return
		}
	})

	t.Run("with valid trim_mode", func(t *testing.T) {
		c, err := common.NewConfigFrom(map[string]interface{}{
			"tokenizer":   "hello %{what}",
			"field":       "message",
			"trim_values": "all",
		})
		if !assert.NoError(t, err) {
			return
		}

		cfg := config{}
		err = c.Unpack(&cfg)
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, trimModeAll, cfg.TrimValues)
	})
}

func TestConfigForDataType(t *testing.T) {
	t.Run("valid data type", func(t *testing.T) {
		c, err := common.NewConfigFrom(map[string]interface{}{
			"tokenizer": "%{value1|integer} %{value2|float} %{value3|boolean} %{value4|long} %{value5|double}",
			"field":     "message",
		})
		if !assert.NoError(t, err) {
			return
		}

		cfg := config{}
		err = c.Unpack(&cfg)
		if !assert.NoError(t, err) {
			return
		}
	})
	t.Run("invalid data type", func(t *testing.T) {
		c, err := common.NewConfigFrom(map[string]interface{}{
			"tokenizer": "%{value1|int} %{value2|short} %{value3|char} %{value4|void} %{value5|unsigned} id=%{id|xyz} status=%{status|abc} msg=\"%{message}\"",
			"field":     "message",
		})
		if !assert.NoError(t, err) {
			return
		}

		cfg := config{}
		err = c.Unpack(&cfg)
		if !assert.Error(t, err) {
			return
		}
	})
	t.Run("missing data type", func(t *testing.T) {
		c, err := common.NewConfigFrom(map[string]interface{}{
			"tokenizer": "%{value1|} %{value2|}",
			"field":     "message",
		})
		if !assert.NoError(t, err) {
			return
		}

		cfg := config{}
		err = c.Unpack(&cfg)
		if !assert.Error(t, err) {
			return
		}
	})
}
