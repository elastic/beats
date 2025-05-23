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

package sys

import (
	"sync"

	"golang.org/x/sys/windows"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

var getCachedANSIDecoder = sync.OnceValue(initANSIDecoder)

func initANSIDecoder() *encoding.Decoder {
	ansiCP := windows.GetACP()
	for _, enc := range charmap.All {
		cm, ok := enc.(*charmap.Charmap)
		if !ok {
			continue
		}
		cmID, _ := cm.ID()
		if uint32(cmID) != ansiCP {
			continue
		}
		return cm.NewDecoder()
	}
	// This should never be reached.
	// If the ANSI Code Page is not found, we will default to
	// Windows1252 Code Page, which is default for ANSI in
	// many regions and corresponds to Western European languages.
	return charmap.Windows1252.NewDecoder()
}

func ANSIBytesToString(enc []byte) (string, error) {
	out, err := getCachedANSIDecoder().Bytes(enc)
	return string(out), err
}
