package kibana

import (
	"github.com/elastic/beats/libbeat/common"
)

func TransformFields(timeFieldName string, title string, commonFields common.Fields) common.MapStr {
	fields := []common.MapStr{}
	fieldFormatMap := common.MapStr{}

	transformFields(commonFields, &fields, fieldFormatMap, "")

	//add some meta fields
	add(common.Field{Path: "_id", Type: "keyword"}, &fields, fieldFormatMap)
	add(common.Field{Path: "_type", Type: "keyword"}, &fields, fieldFormatMap)
	add(common.Field{Path: "_index", Type: "keyword"}, &fields, fieldFormatMap)
	add(common.Field{Path: "_score", Type: "integer"}, &fields, fieldFormatMap)

	return common.MapStr{
		"timeFieldName":  timeFieldName,
		"title":          title,
		"fields":         fields,
		"fieldFormatMap": fieldFormatMap,
	}
}

func transformFields(commonFields common.Fields, fields *[]common.MapStr, fieldFormatMap common.MapStr, path string) {
	for _, f := range commonFields {
		f.Path = f.Name
		if path != "" {
			f.Path = path + "." + f.Name
		}

		if f.Type == "group" {
			if f.Enabled == nil || *f.Enabled {
				transformFields(f.Fields, fields, fieldFormatMap, f.Path)
			}
		} else {
			add(f, fields, fieldFormatMap)

			if f.MultiFields != nil {
				path := f.Path
				for _, mf := range f.MultiFields {
					f.Type = mf.Type
					f.Path = path + "." + mf.Name
					add(f, fields, fieldFormatMap)
				}
			}
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
		"name":     f.Path,
		"type":     typeMapping[f.Type],
		"count":    f.Count,
		"analyzed": analyzed(f),
	}

	if f.Index != nil && *f.Index == false {
		field["index"] = false
		field["readFromDocValues"] = false
		field["analyzed"] = false
		field["searchable"] = false
		field["aggregatable"] = false
	}

	if f.Script != "" {
		field["scripted"] = true
		field["script"] = f.Script
		field["index"] = false
		field["readFromDocValues"] = false
		field["lang"] = "painless"
	}

	//overwrite if values are specifically set,
	//set default value if nil
	if f.Lang != "" {
		field["lang"] = f.Lang
	}
	if f.Scripted != nil {
		field["scripted"] = *f.Scripted
	}
	setVal(field, "index", f.Index, true)
	setVal(field, "searchable", f.Searchable, true)
	setAggregatable(field, f)
	setReadFromDocValues(field, f)

	var format common.MapStr
	if f.Format != "" {
		format = common.MapStr{"id": f.Format}
	}
	addFormatParam(format, "pattern", f.Pattern)
	addFormatParam(format, "input_format", f.InputFormat)
	addFormatParam(format, "output_format", f.OutputFormat)
	addFormatParam(format, "output_precision", f.OutputPrecision)
	addFormatParam(format, "label_template", f.LabelTemplate)
	addFormatParam(format, "url_template", f.UrlTemplate)

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
func setReadFromDocValues(f common.MapStr, c common.Field) {
	//https://www.elastic.co/guide/en/elasticsearch/reference/current/doc-values.html#doc-values
	attr := "readFromDocValues"
	if c.DocValues != nil {
		f[attr] = *c.DocValues
	} else if f["type"] == "string" && f["analyzed"] == true {
		f[attr] = false
	} else if f[attr] == nil {
		f[attr] = true
	}
}

func analyzed(f common.Field) bool {
	if f.Analyzer != "" || f.SearchAnalyzer != "" {
		return true
	}
	return false
}

func addFormatParam(f common.MapStr, key string, val string) {
	if val != "" {
		if f == nil {
			f = common.MapStr{"params": common.MapStr{}}
		} else if f["params"] == nil {
			f["params"] = common.MapStr{}
		}
		f["params"].(common.MapStr)[key] = val
	}
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
		"string":       "string",
		"":             "string",
		"geo_point":    "geo_point",
		"date":         "date",
	}
)
