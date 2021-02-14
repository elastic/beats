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
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/elastic/beats/v7/libbeat/common"
	"io"
	"os"
	"strings"
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

type Decoder struct {
	prependHyphen bool
	lowercaseKeys bool
	xmlDec        *xml.Decoder
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{xmlDec: xml.NewDecoder(r)}
}

func (d *Decoder) PrependHyphen() { d.prependHyphen = true }
func (d *Decoder) LowercaseKeys() { d.lowercaseKeys = true }

var (
	errUnexpectedEnd = errors.New("unexpected end of xml")
)

func (d *Decoder) Decode() (map[string]interface{}, error) {
	_, m, err := d.decode(nil)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(m)
	return m, err
}

type Map map[string]interface{}

func (d *Decoder) decode(attrs []xml.Attr) (string, map[string]interface{}, error) {
	elements := Map{}
	var cdata []byte
	for {
		t, err := d.xmlDec.Token()
		if err != nil {
			if err == io.EOF {
				return "", elements, nil
			}
			return "", nil, err
		}

		switch elem := t.(type) {
		case xml.StartElement:
			cdata, subElements, err := d.decode(elem.Attr)
			if err != nil {
				return "", nil, err
			}

			var add interface{} = subElements
			if len(subElements) == 0 {
				add = cdata
			}

			value := elements[elem.Name.Local]
			switch v := value.(type) {
			case nil:
				elements[elem.Name.Local] = add
			case []interface{}:
				elements[elem.Name.Local] = append(v, add)
			default:
				elements[elem.Name.Local] = []interface{}{v, add}
			}
		case xml.CharData:
			if elemData := bytes.TrimSpace(elem.Copy()); len(elemData) > 0 {
				fmt.Println(string(elemData))
				cdata = elemData
			}
		case xml.EndElement:
			for _, attr := range attrs {
				elements[attr.Name.Local] = attr.Value
			}
			return string(cdata), elements, nil
		}
	}
	return "", nil, errors.New("no end element")
}

func (d *Decoder) addAttributes(attrs []xml.Attr, m map[string]interface{}) {
	for _, attr := range attrs {
		m[attr.Name.Local] = attr.Value
	}
}

func mapGet(key []string, m map[string]interface{}) interface{} {
	v, _ := common.MapStr(m).GetValue(strings.Join(key, "."))
	return v
}

func mapPut(key []string, value interface{}, m map[string]interface{}) {
	common.MapStr(m).Put(strings.Join(key, "."), value)
}
