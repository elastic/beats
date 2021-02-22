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

package macho

import (
	"crypto/md5"
	"debug/macho"
	"encoding/hex"
	"sort"
	"strings"
)

func symhash(machoFile *macho.File) (string, error) {
	if machoFile.Magic == macho.MagicFat {
		return "", nil
	}
	if machoFile.Symtab == nil {
		return "", nil
	}
	if machoFile.Dysymtab == nil {
		return "", nil
	}
	hashed := []string{}
	symbols := machoFile.Symtab.Syms
	for _, symbol := range symbols {
		if symbol.Type&0x0E == 0 {
			hashed = append(hashed, symbol.Name)
		}
	}
	sort.Strings(hashed)
	md5hash := md5.Sum([]byte(strings.Join(hashed, ",")))
	return hex.EncodeToString(md5hash[:]), nil
}
