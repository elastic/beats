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

package autodiscover

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
)

type fakeBuilder struct{}

func (f *fakeBuilder) CreateConfig(event bus.Event) []*common.Config {
	return []*common.Config{common.NewConfig()}
}

func newFakeBuilder(_ *common.Config) (Builder, error) {
	return &fakeBuilder{}, nil
}

func TestBuilderRegistry(t *testing.T) {
	// Add a new builder
	reg := NewRegistry()
	reg.AddBuilder("fake", newFakeBuilder)

	// Check if that builder is available in registry
	b := reg.GetBuilder("fake")
	assert.NotNil(t, b)

	// Generate a config with type fake
	config := BuilderConfig{
		Type: "fake",
	}

	cfg, err := common.NewConfigFrom(&config)

	// Make sure that config building doesn't fail
	assert.Nil(t, err)

	builder, err := reg.BuildBuilder(cfg)
	assert.Nil(t, err)
	assert.NotNil(t, builder)

	// Try to create a config with fake builder and assert length
	// of configs returned is one
	res := builder.CreateConfig(nil)
	assert.Equal(t, len(res), 1)

	builders := Builders{}
	builders = append(builders, builder)

	// Try using builders object for the same as above and expect
	// the same result
	res = builders.GetConfig(nil)
	assert.Equal(t, len(res), 1)
}
