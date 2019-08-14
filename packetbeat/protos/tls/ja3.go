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

package tls

import (
	"crypto/md5"
	"encoding/hex"
	"strconv"
	"strings"
)

func getJa3Fingerprint(hello *helloMessage) (hash string, ja3str string) {

	// build the array of arrays of numbers

	data := make([][]uint16, 5)

	// Version as integer
	data[0] = []uint16{uint16(hello.version.major)*256 + uint16(hello.version.minor)}

	// ciphersuites, in client-supplied priority
	data[1] = make([]uint16, 0)
	for _, suite := range hello.supported.cipherSuites {
		if !isGreaseValue(uint16(suite)) {
			data[1] = append(data[1], uint16(suite))
		}
	}

	// extensions
	data[2] = make([]uint16, len(hello.extensions.InOrder))
	for idx, extid := range hello.extensions.InOrder {
		data[2][idx] = uint16(extid)
	}

	data[3] = extractJa3Array(hello.extensions.Raw[ExtensionSupportedGroups], 2)
	data[4] = extractJa3Array(hello.extensions.Raw[ExtensionEllipticCurvePointsFormats], 1)

	// build the string
	parts := make([]string, len(data))
	for i, arr := range data {
		strNum := make([]string, len(arr))
		for j, num := range arr {
			strNum[j] = strconv.Itoa(int(num))
		}
		parts[i] = strings.Join(strNum, "-")
	}

	ja3str = strings.Join(parts, ",")
	sum := md5.Sum([]byte(ja3str))

	return hex.EncodeToString(sum[:]), ja3str
}

func extractJa3Array(raw []byte, size int) []uint16 {
	if size < 1 || size > 2 {
		return nil
	}
	actual := len(raw)
	if actual < size {
		return nil
	}
	limit := int(raw[0])
	if size == 2 {
		limit = limit*256 + int(raw[1])
	}
	if actual < limit {
		limit = actual
	}
	var array []uint16
	for pos := size; pos <= limit; pos += size {
		value := uint16(raw[pos])
		if size == 2 {
			value = value*256 + uint16(raw[pos+1])
		}
		if !isGreaseValue(value) {
			array = append(array, value)
		}
	}
	return array
}

func isGreaseValue(num uint16) bool {
	hi, lo := byte(num>>8), byte(num)
	return hi == lo && lo&0xf == 0xa
}
