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
			addFormatParam(&format, "inputFormat", f.InputFormat)
			addFormatParam(&format, "outputFormat", f.OutputFormat)
			addFormatParam(&format, "outputPrecision", f.OutputPrecision)
			addFormatParam(&format, "labelTemplate", f.LabelTemplate)
			addFormatParam(&format, "urlTemplate", f.UrlTemplate)
		}
		addFormatParam(&format, "pattern", f.Pattern)
	}

	return field, format
}

func setVal(f common.MapStr, attr string, p *bool, def bool) {
	if p != nil {
		f[attr] = *p
	}
	if f[attr] == nil {
		f[attr] = def
	}
}

func setAggregatable(f common.MapStr, c common.Field) {
	attr := "aggregatable"
	if c.Aggregatable != nil {
		f[attr] = *c.Aggregatable
	} else if c.Type == "text" {
		f[attr] = false
	} else if f[attr] == nil {
		f[attr] = true
	}
}

func getVal(valP *bool, def bool) bool {
	if valP != nil {
		return *valP
	}
	return def
}

func addFormatParam(f *common.MapStr, key string, val string) {
	if val == "" {
		return
	}
	if (*f)["params"] == nil {
		(*f)["params"] = common.MapStr{}
	}
	(*f)["params"].(common.MapStr)[key] = val
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
