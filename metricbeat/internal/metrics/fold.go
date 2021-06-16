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

package metrics

import (
	"github.com/elastic/go-structform"
)

// Fold implements the folder interface for OptUint
func (in *OptUint) Fold(v structform.ExtVisitor) error {
	if in.exists == true {
		value := in.value
		v.OnUint64(value)
	} else {
		v.OnNil()
	}
	return nil
}

// Fold implements the folder interface for OptUint
func (in *OptFloat) Fold(v structform.ExtVisitor) error {
	if in.exists == true {
		value := in.value
		v.OnFloat64(value)
	} else {
		v.OnNil()
	}
	return nil
}
