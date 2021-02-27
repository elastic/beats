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

package dissect

import (
	"fmt"
)

func validate(p *parser) error {
	indirectFields := filterFieldsWith(p.fields, isIndirectField)

	for _, field := range indirectFields {
		found := false
		for _, reference := range p.referenceFields {
			if reference.Key() == field.Key() {
				found = true
				break
			}
		}

		if found == false {
			return fmt.Errorf("missing reference for key '%s'", field.Key())
		}
	}

	return nil
}
