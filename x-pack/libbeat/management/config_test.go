// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestConfigValidate(t *testing.T) {
	tests := map[string]struct {
		config *common.Config
		err    bool
	}{
		"missing access_token": {
			config: common.MustNewConfigFrom(map[string]interface{}{}),
			err:    true,
		},
		"access_token is present": {
			config: common.MustNewConfigFrom(map[string]interface{}{"access_token": "abc1234"}),
			err:    false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			c := defaultConfig()
			err := test.config.Unpack(c)
			if test.err {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
