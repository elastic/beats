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

package kibana

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
)

type fieldsTransformer struct {
	fields                    common.Fields
	transformedFields         []common.MapStr
	transformedFieldFormatMap common.MapStr
	version                   *common.Version
	keys                      common.MapStr
}

func newFieldsTransformer(version *common.Version, fields common.Fields) (*fieldsTransformer, error) {
	if version == nil {
		return nil, errors.New("Version must be given")
	}
	return &fieldsTransformer{
		fields:                    fields,
		version:                   version,
		transformedFields:         []common.MapStr{},
		transformedFieldFormatMap: common.MapStr{},
		keys: common.MapStr{},
	}, nil
}

func (t *fieldsTransformer) transform() (transformed common.MapStr, err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			if err, ok = r.(error); !ok {
				err = fmt.Errorf("Unrecoverable Error %v", r)
			}
		}
	}()

	t.transformFields(t.fields, "")

	// add some meta fields
	truthy := true
	falsy := false
	t.add(common.Field{Path: "_id", Type: "keyword", Index: &falsy, Analyzed: &falsy, DocValues: &falsy, Searchable: &falsy, Aggregatable: &falsy})
	t.add(common.Field{Path: "_type", Type: "keyword", Index: &falsy, Analyzed: &falsy, DocValues: &falsy, Searchable: &truthy, Aggregatable: &truthy})
	t.add(common.Field{Path: "_index", Type: "keyword", Index: &falsy, Analyzed: &falsy, DocValues: &falsy, Searchable: &falsy, Aggregatable: &falsy})
	t.add(common.Field{Path: "_score", Type: "integer", Index: &falsy, Analyzed: &falsy, DocValues: &falsy, Searchable: &falsy, Aggregatable: &falsy})

	transformed = common.MapStr{
		"fields":         t.transformedFields,
		"fieldFormatMap": t.transformedFieldFormatMap,
	}
	return
}

func (t *fieldsTransformer) transformFields(commonFields common.Fields, path string) {
	for _, f := range commonFields {
		f.Path = f.Name
		if path != "" {
			f.Path = path + "." + f.Name
		}

		if t.keys[f.Path] != nil {
			msg := fmt.Sprintf("ERROR: Field <%s> is duplicated. Please update and try again.\n", f.Path)
			panic(errors.New(msg))
		}

		if f.Type == "group" {
			if f.Enabled == nil || *f.Enabled {
				t.transformFields(f.Fields, f.Path)
			}
		} else {
			t.keys[f.Path] = true
			t.add(f)

			if f.MultiFields != nil {
				path := f.Path
				for _, mf := range f.MultiFields {
					f.Type = mf.Type
					f.Path = path + "." + mf.Name
					t.add(f)
				}
			}
		}
	}
}

func (t *fieldsTransformer) add(f common.Field) {
	field, fieldFormat := transformField(t.version, f)
	t.transformedFields = append(t.transformedFields, field)
	if fieldFormat != nil {
		t.transformedFieldFormatMap[field["name"].(string)] = fieldFormat
	}

}

func transformField(version *common.Version, f common.Field) (common.MapStr, common.MapStr) {
	field := common.MapStr{
		"name":         f.Path,
		"count":        f.Count,
		"scripted":     false,
		"indexed":      getVal(f.Index, true),
		"analyzed":     getVal(f.Analyzed, false),
		"doc_values":   getVal(f.DocValues, true),
		"searchable":   getVal(f.Searchable, true),
		"aggregatable": getVal(f.Aggregatable, true),
	}

	if t, ok := typeMapping[f.Type]; ok == true {
		field["type"] = t
	}

	if f.Type == "binary" {
		field["aggregatable"] = false
		field["analyzed"] = false
		field["doc_values"] = getVal(f.DocValues, false)
		field["indexed"] = false
		field["searchable"] = false
	}

	if f.Type == "object" && f.Enabled != nil {
		enabled := getVal(f.Enabled, true)
		field["enabled"] = enabled
		if !enabled {
			field["aggregatable"] = false
			field["analyzed"] = false
			field["doc_values"] = false
			field["indexed"] = false
			field["searchable"] = false
		}
	}

	if f.Type == "text" {
		field["aggregatable"] = false
	}

	if f.Script != "" {
		field["scripted"] = true
		field["script"] = f.Script
		field["lang"] = "painless"
		field["doc_values"] = false
	}

	var format common.MapStr
	if f.Format != "" || f.Pattern != "" {
		format = common.MapStr{}

		if f.Format != "" {
			format["id"] = f.Format
		}
		addParams(&format, version, f)
	}

	return field, format
}

func getVal(valP *bool, def bool) bool {
	if valP != nil {
		return *valP
	}
	return def
}

func addParams(format *common.MapStr, version *common.Version, f common.Field) {
	addFormatParam(format, "pattern", f.Pattern)
	addFormatParam(format, "inputFormat", f.InputFormat)
	addFormatParam(format, "outputFormat", f.OutputFormat)
	addFormatParam(format, "outputPrecision", f.OutputPrecision)
	addFormatParam(format, "labelTemplate", f.LabelTemplate)
	addFormatParam(format, "openLinkInCurrentTab", f.OpenLinkInCurrentTab)
	addVersionedFormatParam(format, version, "urlTemplate", f.UrlTemplate)
}

func addFormatParam(f *common.MapStr, key string, val interface{}) {
	switch val.(type) {
	case string:
		if v := val.(string); v != "" {
			createParam(f)
			(*f)["params"].(common.MapStr)[key] = v
		}
	case *int:
		if v := val.(*int); v != nil {
			createParam(f)
			(*f)["params"].(common.MapStr)[key] = *v
		}
	case *bool:
		if v := val.(*bool); v != nil {
			createParam(f)
			(*f)["params"].(common.MapStr)[key] = *v
		}
	}
}

// takes the highest version where major version <= given version
func addVersionedFormatParam(f *common.MapStr, version *common.Version, key string, val []common.VersionizedString) {
	if len(val) == 0 {
		return
	}
	paramVer, _ := common.NewVersion("0.0.0")
	var paramVal string
	for _, v := range val {
		minVer, err := common.NewVersion(v.MinVersion)
		if err != nil {
			msg := fmt.Sprintf("ERROR: Parameter Version <%s> for <%s> is invalid. Please update and try again.", v.MinVersion, key)
			panic(errors.New(msg))
		}
		if minVer.LessThanOrEqual(true, version) && paramVer.LessThanOrEqual(true, minVer) {
			paramVer = minVer
			paramVal = v.Value
		}
	}
	if paramVal != "" {
		addFormatParam(f, key, paramVal)
	}
}

func createParam(f *common.MapStr) {
	if (*f)["params"] == nil {
		(*f)["params"] = common.MapStr{}
	}
}

var (
	typeMapping = map[string]string{
		"binary":       "binary",
		"half_float":   "number",
		"scaled_float": "number",
		"float":        "number",
		"integer":      "number",
		"long":         "number",
		"short":        "number",
		"byte":         "number",
		"text":         "string",
		"keyword":      "string",
		"":             "string",
		"geo_point":    "geo_point",
		"date":         "date",
		"ip":           "ip",
		"boolean":      "boolean",
	}
)
