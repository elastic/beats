// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package bar

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

func TestBar(t *testing.T) {
	tests := []struct {
		in  interface{}
		err string
	}{
		{
			in: map[string]interface{}{
				"module":     "foo",
				"metricsets": []string{"bar"},
			},
		},
	}

	for i, test := range tests {
		c, err := common.NewConfigFrom(test.in)
		if err != nil {
			t.Fatal(err)
		}

		_, _, err = mb.NewModule(c, mb.Registry)
		if err != nil && test.err == "" {
			t.Errorf("unexpected error in testcase %d: %v", i, err)
			continue
		}
		if test.err != "" {
			if err != nil {
				assert.Contains(t, err.Error(), test.err, "testcase %d", i)
			} else {
				t.Errorf("expected error '%v' in testcase %d", test.err, i)
			}
			continue
		}
	}
}
