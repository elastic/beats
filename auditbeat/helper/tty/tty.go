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

package tty

type TTYType int

const (
	TTYUnknown TTYType = iota
	Pts
	TTY
	TTYConsole
)

const (
	ptsMinMajor     = 136
	ptsMaxMajor     = 143
	ttyMajor        = 4
	consoleMaxMinor = 63
	ttyMaxMinor     = 255
)

type TTYDev struct {
	Minor   uint32
	Major   uint32
	Winsize TTYWinsize
	Termios TTYTermios
}

type TTYWinsize struct {
	Rows uint16
	Cols uint16
}

type TTYTermios struct {
	CIflag uint32
	COflag uint32
	CLflag uint32
	CCflag uint32
}

// interactiveFromTTY returns if this is an interactive tty device.
func InteractiveFromTTY(tty TTYDev) bool {
	return TTYUnknown != GetTTYType(tty.Major, tty.Minor)
}

// getTTYType returns the type of a TTY device based on its major and minor numbers.
func GetTTYType(major uint32, minor uint32) TTYType {
	if major >= ptsMinMajor && major <= ptsMaxMajor {
		return Pts
	}

	if ttyMajor == major {
		if minor <= consoleMaxMinor {
			return TTYConsole
		} else if minor > consoleMaxMinor && minor <= ttyMaxMinor {
			return TTY
		}
	}

	return TTYUnknown
}
