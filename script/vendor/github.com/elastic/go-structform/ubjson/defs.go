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

package ubjson

import "github.com/elastic/go-structform/internal/unsafe"

const (
	noMarker byte = 0

	// value markers
	nullMarker     byte = 'Z'
	noopMarker     byte = 'N'
	trueMarker     byte = 'T'
	falseMarker    byte = 'F'
	int8Marker     byte = 'i'
	uint8Marker    byte = 'U'
	int16Marker    byte = 'I'
	int32Marker    byte = 'l'
	int64Marker    byte = 'L'
	float32Marker  byte = 'd'
	float64Marker  byte = 'D'
	highPrecMarker byte = 'H'
	charMarker     byte = 'C'
	stringMarker   byte = 'S'

	objStartMarker byte = '{'
	objEndMarker   byte = '}'
	arrStartMarker byte = '['
	arrEndMarker   byte = ']'

	countMarker byte = '#'
	typeMarker  byte = '$'
)

func str2Bytes(s string) []byte {
	return unsafe.Str2Bytes(s)
}

func bytes2Str(b []byte) string {
	return unsafe.Bytes2Str(b)
}
