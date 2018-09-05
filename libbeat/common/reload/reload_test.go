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
	obj := reloadable{}
	r := registry{}

	r.Register("my.reloadable", obj)

	assert.Equal(t, obj, r.Get("my.reloadable"))
}

func RegisterReloadableList(t *testing.T) {
	objl := reloadableList{}
	r := registry{}

	r.RegisterList("my.reloadable", objl)

	assert.Equal(t, objl, r.Get("my.reloadable"))
}

func TestRegisterNilFails(t *testing.T) {
	r := registry{}

	err := r.Register("name", nil)
	assert.Error(t, err)

	err = r.RegisterList("name", nil)
	assert.Error(t, err)
}

func TestReRegisterFails(t *testing.T) {
	r := registry{}
	err := r.Register("name", reloadable{})
	assert.NoError(t, err)

	err = r.Register("name", reloadable{})
	assert.Error(t, err)

	err = r.RegisterList("name", reloadableList{})
	assert.NoError(t, err)

	err = r.RegisterList("name", reloadableList{})
	assert.Error(t, err)
}
