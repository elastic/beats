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

package mxj

import (
	"sync"

	"github.com/clbanning/mxj/v2"
)

// The third-party library uses global options. It is unsafe to use the library
// concurrently.
var mutex sync.Mutex

// UnmarshalXML takes a slice of bytes, and returns a map[string]interface{}.
// If the slice is not valid XML, it will return an error.
// This uses the MXJ library compared to the built-in encoding/xml since the latter does not
// support unmarshalling XML to an unknown or empty struct/interface.
//
// Beware that this function acquires a mutux to protect against race conditions
// in the third-party library it wraps.
func UnmarshalXML(body []byte, prepend bool, toLower bool) (map[string]interface{}, error) {
	mutex.Lock()
	defer mutex.Unlock()

	// Disables attribute prefixes and forces all lines to lowercase to meet ECS standards.
	mxj.PrependAttrWithHyphen(prepend)
	mxj.CoerceKeysToLower(toLower)

	xmlObj, err := mxj.NewMapXml(body)
	if err != nil {
		return nil, err
	}

	var out map[string]interface{}
	if err = xmlObj.Struct(&out); err != nil {
		return nil, err
	}
	return out, nil
}
