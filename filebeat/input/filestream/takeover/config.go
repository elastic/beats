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

package takeover

type scanner struct {
	RecursiveGlob bool `config:"recursive_glob"`
}

type prospector struct {
	Scanner scanner `config:"scanner"`
}

type inputConfig struct {
	Type       string     `config:"type"`
	ID         string     `config:"id"`
	Paths      []string   `config:"paths"`
	TakeOver   bool       `config:"take_over"`
	Prospector prospector `config:"prospector"`
}

func defaultInputConfig() inputConfig {
	return inputConfig{
		Type:     "",
		ID:       "",
		Paths:    []string{},
		TakeOver: false,
		Prospector: prospector{
			Scanner: scanner{
				RecursiveGlob: true,
			},
		},
	}
}
