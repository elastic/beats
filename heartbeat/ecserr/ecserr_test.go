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

package ecserr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const typ = "mytype"
const code = "mycode"
const message = "mymessage"

// A var since it's often used as a pointer
var stackTrace = "mystacktrace"

func TestEcsErrNewWithStack(t *testing.T) {
	e := NewECSErrWithStack(typ, code, message, &stackTrace)

	// Ensure that it implments the error interface
	var eErr error = e

	// check that wrapping it still includes the right message
	require.Equal(t, message, eErr.Error())
	require.Equal(t, message, e.Message)

	require.Equal(t, EType(typ), e.Type)
	require.Equal(t, ECode(code), e.Code)
	require.Equal(t, stackTrace, *e.StackTrace)
}

func TestEcsErrNew(t *testing.T) {
	e := NewECSErr(typ, code, message)

	require.Equal(t, message, e.Message)
	require.Equal(t, EType(typ), e.Type)
	require.Equal(t, ECode(code), e.Code)
}
