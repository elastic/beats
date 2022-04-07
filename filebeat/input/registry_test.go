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

package input

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/filebeat/channel"
	"github.com/elastic/beats/v8/libbeat/common"
)

var fakeFactory = func(_ *common.Config, _ channel.Connector, _ Context) (Input, error) {
	return nil, nil
}

func TestAddFactoryEmptyName(t *testing.T) {
	err := Register("", nil)
	if assert.Error(t, err) {
		assert.Equal(t, "Error registering input: name cannot be empty", err.Error())
	}
}

func TestAddNilFactory(t *testing.T) {
	err := Register("name", nil)
	if assert.Error(t, err) {
		assert.Equal(t, "Error registering input 'name': factory cannot be empty", err.Error())
	}
}

func TestAddFactoryTwice(t *testing.T) {
	err := Register("name", fakeFactory)
	if err != nil {
		t.Fatal(err)
	}

	err = Register("name", fakeFactory)
	if assert.Error(t, err) {
		assert.Equal(t, "Error registering input 'name': already registered", err.Error())
	}
}

func TestGetFactory(t *testing.T) {
	f, err := GetFactory("name")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, f)
}

func TestGetNonExistentFactory(t *testing.T) {
	f, err := GetFactory("noSuchFactory")
	assert.Nil(t, f)
	if assert.Error(t, err) {
		assert.Equal(t, "Error creating input. No such input type exist: 'noSuchFactory'", err.Error())
	}
}
