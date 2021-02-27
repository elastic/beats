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

package javascript

var sessionHooks = map[string]SessionHook{}

// SessionHook is a function that get invoked when each new Session is created.
type SessionHook func(s Session)

// AddSessionHook registers a SessionHook that gets invoked for each new Session.
func AddSessionHook(name string, mod SessionHook) {
	sessionHooks[name] = mod
}
