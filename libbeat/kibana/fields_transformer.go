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

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/mapping"
)

var v640 = common.MustNewVersion("6.4.0")

type fieldsTransformer struct {
	fields                    mapping.Fields
	transformedFields         []common.MapStr
	transformedFieldFormatMap common.MapStr
	version                   *common.Version
	keys                      map[string]int
	migration                 bool
}

func newFieldsTransformer(version *common.Version, fields mapping.Fields, migration bool) (*fieldsTransformer, error) {
	if version == nil {
		return nil, errors.New("Version must be given")
	}
	return &fieldsTransformer{
		fields:                    fields,
		version:                   version,
		transformedFields:         []common.MapStr{},
		transformedFieldFormatMap: common.MapStr{},
		keys:                      map[string]int{},
		migration:                 migration,
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
	t.add(mapping.Field{Path: "_id", Type: "keyword", Index: &falsy, Analyzed: &falsy, DocValues: &falsy, Searchable: &falsy, Aggregatable: &falsy})
	t.add(mapping.Field{Path: "_type", Type: "keyword", Index: &falsy, Analyzed: &falsy, DocValues: &falsy, Searchable: &truthy, Aggregatable: &truthy})
	t.add(mapping.Field{Path: "_index", Type: "keyword", Index: &falsy, Analyzed: &falsy, DocValues: &falsy, Searchable: &falsy, Aggregatable: &falsy})
	t.add(mapping.Field{Path: "_score", Type: "integer", Index: &falsy, Analyzed: &falsy, DocValues: &falsy, Searchable: &falsy, Aggregatable: &falsy})

	transformed = common.MapStr{
		"fields":         t.transformedFields,
		"fieldFormatMap": t.transformedFieldFormatMap,
	}
	return
}

func (t *fieldsTransformer) transformFields(commonFields mapping.Fields, path string) {
	for _, f := range commonFields {
		f.Path = f.Name
		if path != "" {
			f.Path = path + "." + f.Name
		}

		if f.Type == "group" {
			if f.Enabled == nil || *f.Enabled {
				t.transformFields(f.Fields, f.Path)
			}
		} else {
			if f.Type == "alias" {
				if t.version.LessThan(v640) {
					continue
				}
				// Only adds migration aliases if migration is enabled
				if f.MigrationAlias && !t.migration {
					continue
				}
				if ff := t.fields.GetField(f.AliasPath); ff != nil {
					// copy the field, keep
					path := f.Path
					name := f.Name
					f = *ff
					f.Path = path
					f.Name = name
				}
			}
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

func (t *fieldsTransformer) update(target *common.MapStr, override mapping.Field) error {
	field, _ := transformField(t.version, override)
	if override.Type == "" || (*target)["type"] == field["type"] {
		target.Update(field)
		if !override.Overwrite {
			// compatible duplication
			return fmt.Errorf("field <%s> is duplicated, remove it or set 'overwrite: true', %+v, %+v", override.Path, override, field)
		}
		return nil
	}
	// incompatible duplication
	return fmt.Errorf("field <%s> is duplicated", override.Path)
}

func (t *fieldsTransformer) add(f mapping.Field) {
	if idx := t.keys[f.Path]; idx > 0 {
		target := &t.transformedFields[idx-1] // 1-indexed
		if err := t.update(target, f); err != nil {
			panic(err)
		}
		return
	}

	field, fieldFormat := transformField(t.version, f)
	t.transformedFields = append(t.transformedFields, field)
	t.keys[f.Path] = len(t.transformedFields) // 1-index
	if fieldFormat != nil {
		t.transformedFieldFormatMap[field["name"].(string)] = fieldFormat
	}
}

func transformField(version *common.Version, f mapping.Field) (common.MapStr, common.MapStr) {
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

func addParams(format *common.MapStr, version *common.Version, f mapping.Field) {
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
func addVersionedFormatParam(f *common.MapStr, version *common.Version, key string, val []mapping.VersionizedString) {
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
		"ip_range":     "ip_range",
		"boolean":      "boolean",
	}
)
