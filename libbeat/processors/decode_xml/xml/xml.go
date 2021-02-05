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

package xml

import (
	"github.com/clbanning/mxj/v2"
)

// UnmarshalXML takes a slice of bytes, and returns a map[string]interface{}.
// If the slice is not valid XML, it will return an error.
func UnmarshalXML(body []byte, prepend bool, toLower bool) (obj map[string]interface{}, err error) {
	var xmlobj mxj.Map
	// Disables attribute prefixes and forces all lines to lowercase to meet ECS standards
	mxj.PrependAttrWithHyphen(prepend)
	mxj.CoerceKeysToLower(toLower)

	xmlobj, err = mxj.NewMapXml(body)
	if err != nil {
		return nil, err
	}

	err = xmlobj.Struct(&obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
