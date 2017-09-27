package kibana

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
)

func TransformFields(timeFieldName string, title string, commonFields common.Fields) common.MapStr {
	fields := []common.MapStr{}
	fieldFormatMap := common.MapStr{}
	keys := common.MapStr{}

	transformFields(keys, commonFields, &fields, fieldFormatMap, "")

	// add some meta fields
	truthy := true
	falsy := false
	add(common.Field{Path: "_id", Type: "keyword", Index: &falsy, Analyzed: &falsy, DocValues: &falsy, Searchable: &falsy, Aggregatable: &falsy}, &fields, fieldFormatMap)
	add(common.Field{Path: "_type", Type: "keyword", Index: &falsy, Analyzed: &falsy, DocValues: &falsy, Searchable: &truthy, Aggregatable: &truthy}, &fields, fieldFormatMap)
	add(common.Field{Path: "_index", Type: "keyword", Index: &falsy, Analyzed: &falsy, DocValues: &falsy, Searchable: &falsy, Aggregatable: &falsy}, &fields, fieldFormatMap)
	add(common.Field{Path: "_score", Type: "integer", Index: &falsy, Analyzed: &falsy, DocValues: &falsy, Searchable: &falsy, Aggregatable: &falsy}, &fields, fieldFormatMap)

	return common.MapStr{
		"timeFieldName":  timeFieldName,
		"title":          title,
		"fields":         fields,
		"fieldFormatMap": fieldFormatMap,
	}
}

func transformFields(keys common.MapStr, commonFields common.Fields, fields *[]common.MapStr, fieldFormatMap common.MapStr, path string) {
	for _, f := range commonFields {
		f.Path = f.Name
		if path != "" {
			f.Path = path + "." + f.Name
		}

		if keys[f.Path] != nil {
			msg := fmt.Sprintf("ERROR: Field <%s> is duplicated. Please update and try again.", f.Path)
			panic(errors.New(msg))
		}

		if f.Type == "group" {
			transformFields(keys, f.Fields, fields, fieldFormatMap, f.Path)
		} else {
			// set default values (as done in python script)
			keys[f.Path] = true

			truthy := true
			falsy := false
			f.Index = &truthy
			f.Analyzed = &falsy
			f.DocValues = &truthy
			f.Searchable = &truthy
			f.Aggregatable = &truthy
			add(f, fields, fieldFormatMap)
		}
	}
}

func add(f common.Field, fields *[]common.MapStr, fieldFormatMap common.MapStr) {
	field, fieldFormat := transformField(f)
	*fields = append(*fields, field)
	if fieldFormat != nil {
		fieldFormatMap[field["name"].(string)] = fieldFormat
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
