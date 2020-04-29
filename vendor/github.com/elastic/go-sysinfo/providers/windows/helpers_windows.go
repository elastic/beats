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

package windows

import (
	"fmt"

	syswin "golang.org/x/sys/windows"
)

// sidToString wraps the `String()` functions used to return SID strings in golang.org/x/sys
// These can return an error or no error, depending on the release.
func sidToString(strFunc *syswin.SID) (string, error) {
	switch sig := (interface{})(strFunc).(type) {
	case fmt.Stringer:
		return sig.String(), nil
	case errString:
		return sig.String()
	default:
		return "", fmt.Errorf("missing or unexpected String() function signature for %#v", sig)
	}
}

type errString interface {
	String() (string, error)
}
