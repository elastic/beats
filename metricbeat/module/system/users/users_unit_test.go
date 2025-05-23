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

//go:build linux && !integration

package users

import (
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/stretchr/testify/assert"
)

func TestFormatSessionList(t *testing.T) {
	testIn := [][]dbus.Variant{
		{dbus.MakeVariant("6"), dbus.MakeVariant(uint32(1000)), dbus.MakeVariant("user"), dbus.MakeVariant(""), dbus.MakeVariant(dbus.ObjectPath("/path/to/object"))},
	}

	goodOut := []loginSession{{
		ID:   "6",
		UID:  uint32(1000),
		User: "user",
		Seat: "",
		Path: dbus.ObjectPath("/path/to/object"),
	},
	}

	output, err := formatSessionList(testIn)
	assert.NoError(t, err)
	assert.Equal(t, goodOut, output)

}
