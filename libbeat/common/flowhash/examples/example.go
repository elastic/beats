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

package flowhash_test

import (
	"fmt"
	"net"

	"github.com/elastic/beats/v8/libbeat/common/flowhash"
)

// ExampleCommunityIDHash shows example usage for flowhash.CommunityID.Hash()
func ExampleCommunityIDHash() {
	flow := flowhash.Flow{
		SourceIP:        net.ParseIP("10.1.2.3"),
		DestinationIP:   net.ParseIP("8.8.8.8"),
		SourcePort:      63521,
		DestinationPort: 53,
		Protocol:        17,
	}
	fmt.Println(flowhash.CommunityID.Hash(flow))
	// Output: 1:R7iR6vkxw+jaz3wjDfWMWooBdfc=
}
