// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
)

func TestHasNonNumericFloat(t *testing.T) {
	type caseKey struct {
		key           string
		expectedFound bool
		expectedErr   bool
	}
	cases := []struct {
		title string
		event common.MapStr
		keys  []caseKey
	}{
		{
			title: "Empty event",
			event: common.MapStr{},
			keys: []caseKey{
				{"", false, true},
				{"somekey", false, true},
			},
		},
		{
			title: "Event with non-numeric values",
			event: common.MapStr{
				"someobject": common.MapStr{
					"inf":    math.Inf(1),
					"nan":    math.NaN(),
					"number": int64(42),
					"float":  float64(42),
				},
			},
			keys: []caseKey{
				{"", false, true},
				{"someobject", false, false},
				{"someobject.inf", true, false},
				{"someobject.nan", true, false},
				{"someobject.number", false, false},
				{"someobject.float", false, false},
				{"someobject.notexists", false, true},
			},
		},
	}

	for _, c := range cases {
		for _, k := range c.keys {
			t.Run(c.title+"/"+k.key, func(t *testing.T) {
				found, err := HasNonNumericFloat(c.event, k.key)
				assert.Equal(t, k.expectedFound, found, "key has numeric float")
				if k.expectedErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	}
}
