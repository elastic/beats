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

package consumergroup

type nameSet map[string]struct{}

func makeNameSet(strings ...string) nameSet {
	if len(strings) == 0 {
		return nil
	}

	set := nameSet{}
	for _, s := range strings {
		set[s] = struct{}{}
	}
	return set
}

func (s nameSet) has(name string) bool {
	if s == nil {
		return true
	}

	_, ok := s[name]
	return ok
}

func (s nameSet) pred() func(string) bool {
	if s == nil || len(s) == 0 {
		return nil
	}
	return s.has
}
