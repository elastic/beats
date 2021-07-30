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

package windows

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandLineToArgv(t *testing.T) {
	cases := []struct {
		cmd  string
		args []string
	}{
		{
			cmd:  ``,
			args: nil,
		},
		{
			cmd:  ` `,
			args: nil,
		},
		{
			cmd:  "\t",
			args: nil,
		},
		{
			cmd:  `test`,
			args: []string{`test`},
		},
		{
			cmd:  `test a b c`,
			args: []string{`test`, `a`, `b`, `c`},
		},
		{
			cmd:  `test "`,
			args: []string{`test`, ``},
		},
		{
			cmd:  `test ""`,
			args: []string{`test`, ``},
		},
		{
			cmd:  `test """`,
			args: []string{`test`, `"`},
		},
		{
			cmd:  `test "" a`,
			args: []string{`test`, ``, `a`},
		},
		{
			cmd:  `test "123"`,
			args: []string{`test`, `123`},
		},
		{
			cmd:  `test \"123\"`,
			args: []string{`test`, `"123"`},
		},
		{
			cmd:  `test \"123 456\"`,
			args: []string{`test`, `"123`, `456"`},
		},
		{
			cmd:  `test \\"`,
			args: []string{`test`, `\`},
		},
		{
			cmd:  `test \\\"`,
			args: []string{`test`, `\"`},
		},
		{
			cmd:  `test \\\\\"`,
			args: []string{`test`, `\\"`},
		},
		{
			cmd:  `test \\\"x`,
			args: []string{`test`, `\"x`},
		},
		{
			cmd:  `test """"\""\\\"`,
			args: []string{`test`, `""\"`},
		},
		{
			cmd:  `"cmd line" abc`,
			args: []string{`cmd line`, `abc`},
		},
		{
			cmd:  `test \\\\\""x"""y z`,
			args: []string{`test`, `\\"x"y z`},
		},
		{
			cmd:  "test\tb\t\"x\ty\"",
			args: []string{`test`, `b`, "x\ty"},
		},
		{
			cmd: `"C:\Program Files (x86)\Steam\bin\cef\cef.win7x64\steamwebhelper.exe" "-lang=en_US" "-cachedir=C:\Users\jimmy\AppData\Local\Steam\htmlcache" "-steampid=796" "-buildid=1546909276" "-steamid=0" "-steamuniverse=Dev" "-clientui=C:\Program Files (x86)\Steam\clientui" --disable-spell-checking --disable-out-of-process-pac --enable-blink-features=ResizeObserver,Worklet,AudioWorklet --disable-features=TouchpadAndWheelScrollLatching,AsyncWheelEvents --enable-media-stream --disable-smooth-scrolling --num-raster-threads=4 --enable-direct-write "--log-file=C:\Program Files (x86)\Steam\logs\cef_log.txt"`,
			args: []string{
				`C:\Program Files (x86)\Steam\bin\cef\cef.win7x64\steamwebhelper.exe`,
				`-lang=en_US`,
				`-cachedir=C:\Users\jimmy\AppData\Local\Steam\htmlcache`,
				`-steampid=796`,
				`-buildid=1546909276`,
				`-steamid=0`,
				`-steamuniverse=Dev`,
				`-clientui=C:\Program Files (x86)\Steam\clientui`,
				`--disable-spell-checking`,
				`--disable-out-of-process-pac`,
				`--enable-blink-features=ResizeObserver,Worklet,AudioWorklet`,
				`--disable-features=TouchpadAndWheelScrollLatching,AsyncWheelEvents`,
				`--enable-media-stream`,
				`--disable-smooth-scrolling`,
				`--num-raster-threads=4`,
				`--enable-direct-write`,
				`--log-file=C:\Program Files (x86)\Steam\logs\cef_log.txt`,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.cmd, func(t *testing.T) {
			assert.Equal(t, tc.args, SplitCommandLine(tc.cmd))
		})
	}
}
