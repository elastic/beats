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

package feature

import "fmt"

// Details minimal information that you must provide when creating a feature.
type Details struct {
	Name       string
	Stability  Stability
	Deprecated bool
	Info       string // short info string
	Doc        string // long doc string
}

func (d Details) String() string {
	fmtStr := "name: %s, description: %s (%s)"
	if d.Deprecated {
		fmtStr = "name: %s, description: %s (deprecated, %s)"
	}
	return fmt.Sprintf(fmtStr, d.Name, d.Info, d.Stability)
}

// MakeDetails return the minimal information a new feature must provide.
func MakeDetails(fullName string, doc string, stability Stability) Details {
	return Details{Name: fullName, Info: doc, Stability: stability}
}
