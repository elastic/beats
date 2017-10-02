package kibana

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
)

type Transformer struct {
	fields                    common.Fields
	transformedFields         []common.MapStr
	transformedFieldFormatMap common.MapStr
	timeFieldName             string
	title                     string
	keys                      common.MapStr
}

func NewTransformer(timeFieldName, title string, fields common.Fields) *Transformer {
	return &Transformer{
		fields:                    fields,
		timeFieldName:             timeFieldName,
		title:                     title,
		transformedFields:         []common.MapStr{},
		transformedFieldFormatMap: common.MapStr{},
		keys: common.MapStr{},
	}
}

func (t *Transformer) TransformFields() common.MapStr {
	t.transformFields(t.fields, "")

	// add some meta fields
	truthy := true
	falsy := false
	t.add(common.Field{Path: "_id", Type: "keyword", Index: &falsy, Analyzed: &falsy, DocValues: &falsy, Searchable: &falsy, Aggregatable: &falsy})
	t.add(common.Field{Path: "_type", Type: "keyword", Index: &falsy, Analyzed: &falsy, DocValues: &falsy, Searchable: &truthy, Aggregatable: &truthy})
	t.add(common.Field{Path: "_index", Type: "keyword", Index: &falsy, Analyzed: &falsy, DocValues: &falsy, Searchable: &falsy, Aggregatable: &falsy})
	t.add(common.Field{Path: "_score", Type: "integer", Index: &falsy, Analyzed: &falsy, DocValues: &falsy, Searchable: &falsy, Aggregatable: &falsy})

	return common.MapStr{
		"timeFieldName":  t.timeFieldName,
		"title":          t.title,
		"fields":         t.transformedFields,
		"fieldFormatMap": t.transformedFieldFormatMap,
	}
}

func (t *Transformer) transformFields(commonFields common.Fields, path string) {
	for _, f := range commonFields {
		f.Path = f.Name
		if path != "" {
			f.Path = path + "." + f.Name
		}

		if t.keys[f.Path] != nil {
			msg := fmt.Sprintf("ERROR: Field <%s> is duplicated. Please update and try again.", f.Path)
			panic(errors.New(msg))
		}

		if f.Type == "group" {
			if f.Enabled == nil || *f.Enabled {
				t.transformFields(f.Fields, f.Path)
			}
		} else {
			// set default values (as done in python script)
			t.keys[f.Path] = true

			truthy := true
			falsy := false
			f.Index = &truthy
			f.Analyzed = &falsy
			f.DocValues = &truthy
			f.Searchable = &truthy
			f.Aggregatable = &truthy
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

func (t *Transformer) add(f common.Field) {
	field, fieldFormat := transformField(f)
	t.transformedFields = append(t.transformedFields, field)
	if fieldFormat != nil {
		t.transformedFieldFormatMap[field["name"].(string)] = fieldFormat
	}

}

func transformField(f common.Field) (common.MapStr, common.MapStr) {
	field := common.MapStr{
		"name":         f.Path,
		"count":        0,
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

	if f.Type == "text" {
		field["aggregatable"] = false
	}

	var format common.MapStr
	if f.Format != "" || f.Pattern != "" {
		format = common.MapStr{}

		if f.Format != "" {
			format["id"] = f.Format
			if f.InputFormat != "" {
				format["params"] = common.MapStr{"inputFormat": f.InputFormat}
			}
		}
		if f.Pattern != "" {
			format["params"] = common.MapStr{"pattern": f.Pattern}
		}
	}

	return field, format
}

func getVal(valP *bool, def bool) bool {
	if valP != nil {
		return *valP
	}
	return def
}

var (
	typeMapping = map[string]string{
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
	}
)
