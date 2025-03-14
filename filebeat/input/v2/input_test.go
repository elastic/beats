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

package v2

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// TestNewContextParameters ensures when new fields are added to the v2.Context
// they also are added to the constructor, or the decision of not doing so is
// explicit.
func TestNewContextParameters(t *testing.T) {
	ctx := NewContext(
		"TestNewContextParameters",
		"TestNewContextParameters",
		"test-input",
		beat.Info{},
		context.Background(),
		noopStatusReporter{},
		monitoring.NewRegistry(),
		logp.NewLogger("test"),
	)

	v := reflect.ValueOf(ctx)
	typeOfCtx := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := typeOfCtx.Field(i).Name

		// ignore unexported fields
		if !field.CanSet() {
			continue
		}

		assert.Falsef(t, field.IsZero(),
			"v2.Context field %s was not set by the constructor. A new field"+
				"might have been added, please consider if you need to change "+
				"the constructor or to skip the field in this test",
			fieldName)
	}
}

type noopStatusReporter struct{}

func (n noopStatusReporter) UpdateStatus(status.Status, string) {
}
