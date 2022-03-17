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

package ecs

// The user_agent fields normally come from a browser request.
// They often show up in web service logs coming from the parsed user agent
// string.
type UserAgent struct {
	// Unparsed user_agent string.
	Original string `ecs:"original"`

	// Name of the user agent.
	Name string `ecs:"name"`

	// Version of the user agent.
	Version string `ecs:"version"`

	// Name of the device.
	DeviceName string `ecs:"device.name"`
}
