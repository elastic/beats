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

// An autonomous system (AS) is a collection of connected Internet Protocol
// (IP) routing prefixes under the control of one or more network operators on
// behalf of a single administrative entity or domain that presents a common,
// clearly defined routing policy to the internet.
type AS struct {
	// Unique number allocated to the autonomous system. The autonomous system
	// number (ASN) uniquely identifies each network on the Internet.
	Number int64 `ecs:"number"`

	// Organization name.
	OrganizationName string `ecs:"organization.name"`
}
