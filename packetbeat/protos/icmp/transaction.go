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

package icmp

import "time"

type icmpTransaction struct {
	ts    time.Time // timestamp of the first packet
	tuple icmpTuple
	notes []string

	request  *icmpMessage
	response *icmpMessage
}

func (t *icmpTransaction) HasError() bool {
	return t.request == nil ||
		(t.request != nil && isError(&t.tuple, t.request)) ||
		(t.response != nil && isError(&t.tuple, t.response)) ||
		(t.request != nil && t.response == nil && requiresCounterpart(&t.tuple, t.request))
}
