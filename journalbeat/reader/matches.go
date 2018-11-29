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

//+build linux,cgo

package reader

import (
	"fmt"
	"strings"
)

func (r *Reader) addMatches() error {
	for _, m := range r.config.Matches {
		elems := strings.Split(m, "=")
		if len(elems) != 2 {
			return fmt.Errorf("invalid match format: %s", m)
		}

		var p string
		for journalKey, eventField := range journaldEventFields {
			if elems[0] == eventField.name {
				p = journalKey + "=" + elems[1]
			}
		}

		// pass custom fields as is
		if p == "" {
			p = m
		}

		r.logger.Debug("journal", "Added matcher expression: %s", p)

		err := r.journal.AddMatch(p)
		if err != nil {
			return fmt.Errorf("error adding match to journal: %+v", err)
		}

		err = r.journal.AddDisjunction()
		if err != nil {
			return fmt.Errorf("error adding disjunction to journal: %+v", err)
		}
	}
	return nil
}
