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

//go:build windows
// +build windows

package file

var globTests = []globTest{
	{
		"*",
		[]string{
			"foo",
		},
	},
	{
		"foo\\*",
		[]string{
			"foo\\bar",
		},
	},
	{
		"foo/*",
		[]string{
			"foo\\bar",
		},
	},
	{
		"*\\*",
		[]string{
			"foo\\bar",
		},
	},
	{
		"**",
		[]string{
			"",
			"foo",
			"foo\\bar",
			"foo\\bar\\baz",
			"foo\\bar\\baz\\qux",
		},
	},
	{
		"foo**",
		[]string{
			"foo",
		},
	},
	{
		"foo\\**",
		[]string{
			"foo",
			"foo\\bar",
			"foo\\bar\\baz",
			"foo\\bar\\baz\\qux",
			"foo\\bar\\baz\\qux\\quux",
		},
	},
	{
		"foo\\**\\baz",
		[]string{
			"foo\\bar\\baz",
		},
	},
	{
		"foo/**\\baz",
		[]string{
			"foo\\bar\\baz",
		},
	},
	{
		"foo\\**\\bazz",
		[]string{},
	},
	{
		"foo\\**\\bar",
		[]string{
			"foo\\bar",
		},
	},
	{
		"foo\\\\bar",
		[]string{
			"foo\\bar",
		},
	},
}

var globPatternsTests = []globPatternsTest{
	{
		"C:\\foo\\**\\bar",
		[]string{"C:\\foo\\bar", "C:\\foo\\*\\bar", "C:\\foo\\*\\*\\bar"},
		false,
	},
}
