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

//go:build linux

package security

import (
	"os/user"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetNodeProcAttrUnknownUser(t *testing.T) {
	const name = "no-such-user-heartbeat-test"

	err := setNodeProcAttr(name)

	require.Error(t, err, "looking up an unknown user must fail")
	assert.ErrorContains(t, err, "could not lookup '"+name+"'", "the error must name the requested user")
	assert.NotContains(t, err.Error(), "%!", "the error must not contain a formatting artifact such as a nil *user.User")
	var unknown user.UnknownUserError
	assert.ErrorAs(t, err, &unknown, "the original lookup error must stay wrapped")
}
