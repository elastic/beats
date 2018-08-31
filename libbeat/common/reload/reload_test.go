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

package reload

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type nonreloadable struct{}
type reloadable struct{}
type reloadableList struct{}

func (reloadable) Reload(config *ConfigWithMeta) error       { return nil }
func (reloadableList) Reload(config []*ConfigWithMeta) error { return nil }

func RegisterReloadable(t *testing.T) {
	r := reloadable{}

	MustRegister("my.reloadable", reloadable{})

	assert.Equal(t, r, Get("my.reloadable"))
}

func RegisterReloadableList(t *testing.T) {
	r := reloadableList{}

	MustRegisterList("my.reloadable", reloadableList{})

	assert.Equal(t, r, Get("my.reloadable"))
}

func TestRegisterNilFails(t *testing.T) {
	assert.Panics(t, func() {
		MustRegister("name", nil)
	})

	assert.Panics(t, func() {
		MustRegisterList("name", nil)
	})
}

func TestReRegisterFails(t *testing.T) {
	assert.Panics(t, func() {
		MustRegister("name", reloadable{})
		MustRegister("name", reloadable{})
	})

	assert.Panics(t, func() {
		MustRegisterList("mylist", reloadableList{})
		MustRegisterList("mylist", reloadableList{})
	})
}
