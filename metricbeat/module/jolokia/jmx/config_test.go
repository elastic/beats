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

package jmx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanonicalMBeanName(t *testing.T) {
	cases := []struct {
		mbean    string
		expected string
		ok       bool
	}{
		{
			mbean: ``,
			ok:    false,
		},
		{
			mbean: `type=Runtime`,
			ok:    false,
		},
		{
			mbean: `java.lang`,
			ok:    false,
		},
		{
			mbean: `java.lang:`,
			ok:    false,
		},
		{
			mbean: `java.lang:type=Runtime,name`,
			ok:    false,
		},
		{
			mbean:    `java.lang:type=Runtime`,
			expected: `java.lang:type=Runtime`,
			ok:       true,
		},
		{
			mbean:    `java.lang:name=Foo,type=Runtime`,
			expected: `java.lang:name=Foo,type=Runtime`,
			ok:       true,
		},
		{
			mbean:    `java.lang:type=Runtime,name=Foo`,
			expected: `java.lang:name=Foo,type=Runtime`,
			ok:       true,
		},
		{
			mbean:    `java.lang:type=Runtime,name=Foo*`,
			expected: `java.lang:name=Foo*,type=Runtime`,
			ok:       true,
		},
		{
			mbean:    `java.lang:type=Runtime,name=*`,
			expected: `java.lang:name=*,type=Runtime`,
			ok:       true,
		},
		{
			mbean:    `java.lang:type=Runtime,name="foo,bar"`,
			expected: `java.lang:name="foo,bar",type=Runtime`,
			ok:       true,
		},
		{
			mbean:    `Catalina:type=RequestProcessor,worker="http-nio-8080",name=HttpRequest1`,
			expected: `Catalina:name=HttpRequest1,type=RequestProcessor,worker="http-nio-8080"`,
			ok:       true,
		},
	}

	for _, c := range cases {
		canonical, err := canonicalizeMBeanName(c.mbean)
		if c.ok {
			assert.NoError(t, err, "failed parsing for: "+c.mbean)
			assert.Equal(t, c.expected, canonical, "mbean: "+c.mbean)
		} else {
			assert.Error(t, err, "should have failed for: "+c.mbean)
		}
	}
}
