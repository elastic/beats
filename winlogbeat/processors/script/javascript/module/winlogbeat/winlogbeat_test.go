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

// +build windows

package winlogbeat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const quotedCommandLine = `"C:\Program Files (x86)\Steam\bin\cef\cef.win7x64\steamwebhelper.exe" "-lang=en_US" "-cachedir=C:\Users\jimmy\AppData\Local\Steam\htmlcache" "-steampid=796" "-buildid=1546909276" "-steamid=0" "-steamuniverse=Dev" "-clientui=C:\Program Files (x86)\Steam\clientui" --disable-spell-checking --disable-out-of-process-pac --enable-blink-features=ResizeObserver,Worklet,AudioWorklet --disable-features=TouchpadAndWheelScrollLatching,AsyncWheelEvents --enable-media-stream --disable-smooth-scrolling --num-raster-threads=4 --enable-direct-write "--log-file=C:\Program Files (x86)\Steam\logs\cef_log.txt"`

func TestSplitCommandLine(t *testing.T) {
	args := SplitCommandLine(quotedCommandLine)

	for _, a := range args {
		t.Log(a)
	}

	expected := []string{
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
	}
	assert.Equal(t, expected, args)
}
