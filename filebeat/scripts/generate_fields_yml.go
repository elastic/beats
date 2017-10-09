package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"reflect"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	pipelinePath  = "../module/%s/%s/ingest/pipeline.json"
	fieldsYmlPath = "../module/%s/%s/_meta/fields.yml"
)

var (
	types = map[string]string{
		"group":           "group",
		"DATA":            "text",
		"GREEDYDATA":      "text",
		"GREEDYMULTILINE": "text",
		"HOSTNAME":        "keyword",
		"IPHOST":          "keyword",
		"IPORHOST":        "keyword",
		"LOGLEVEL":        "keyword",
		"MULTILINEQUERY":  "text",
		"NUMBER":          "long",
		"POSINT":          "long",
		"SYSLOGHOST":      "keyword",
		"SYSLOGTIMESTAMP": "text",
		"TIMESTAMP":       "text",
		"USERNAME":        "keyword",
		"WORD":            "keyword",
	}
)

type Pipeline struct {
	Description string                   `json:"description"`
	Processors  []map[string]interface{} `json:"processors"`
	OnFailure   interface{}              `json:"on_failure"`
}

type Field struct {
	Type     string
	Elements []string
}

type FieldYml struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description,omitempty"`
	Example     string      `yaml:"example,omitempty"`
	Type        string      `yaml:"type,omitempty"`
	Fields      []*FieldYml `yaml:"fields,omitempty"`
}

func NewFieldYml(name, typeName string, noDoc bool) *FieldYml {
	if noDoc {
		return &FieldYml{
			Name: name,
			Type: typeName,
		}
	}

	return &FieldYml{
		Name:        name,
		Type:        typeName,
		Description: "Please add description",
		Example:     "Please add example",
	}
}

func NewField(lp string) Field {
	lp = lp[1 : len(lp)-1]
	ee := strings.Split(lp, ":")
	e := strings.Split(ee[1], ".")
	return Field{
		Type:     ee[0],
		Elements: e,
	}
}

func readPipeline(module, fileset string) (*Pipeline, error) {
	pp := fmt.Sprintf(pipelinePath, module, fileset)
	r, err := ioutil.ReadFile(pp)
	if err != nil {
		return nil, err
	}

	var p Pipeline
	err = json.Unmarshal(r, &p)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func addNewField(fs []Field, f Field) []Field {
	for _, ff := range fs {
		if reflect.DeepEqual(ff, f) {
			return fs
		}
	}
	return append(fs, f)
}

func getElementsFromPatterns(patterns []string) ([]Field, error) {
	r, err := regexp.Compile("{[\\.\\w\\:]*}")
	if err != nil {
		return nil, err
	}

	fs := make([]Field, 0)
	for _, lp := range patterns {
		pp := r.FindAllString(lp, -1)
		for _, p := range pp {
			f := NewField(p)
			fs = addNewField(fs, f)
		}

	}
	return fs, nil
}

func accumulatePatterns(grok interface{}) ([]string, error) {
	for k, v := range grok.(map[string]interface{}) {
		if k == "patterns" {
			vs := v.([]interface{})
			p := make([]string, 0)
			for _, s := range vs {
				p = append(p, s.(string))
			}
			return p, nil
		}
	}
	return nil, fmt.Errorf("No patterns in pipeline")
}

func getPatternsFromProcessors(p []map[string]interface{}) ([]string, error) {
	for _, e := range p {
		if ee, ok := e["grok"]; ok {
			return accumulatePatterns(ee)
		}
	}
	return nil, fmt.Errorf("No patterns in pipeline")
}

func getFieldByName(f []*FieldYml, name string) *FieldYml {
	for _, ff := range f {
		if ff.Name == name {
			return ff
		}
	}
	return nil
}

func insertLastField(f []*FieldYml, name, typeName string, noDoc bool) []*FieldYml {
	ff := getFieldByName(f, name)
	if ff != nil {
		return f
	}

	nf := NewFieldYml(name, types[typeName], noDoc)
	return append(f, nf)
}

func insertGroup(out []*FieldYml, field Field, index, count int, noDoc bool) []*FieldYml {
	g := getFieldByName(out, field.Elements[index])
	if g != nil {
		g.Fields = generateField(g.Fields, field, index+1, count, noDoc)
		return out
	} else {
		groupFields := make([]*FieldYml, 0)
		groupFields = generateField(groupFields, field, index+1, count, noDoc)
		group := NewFieldYml(field.Elements[index], "group", noDoc)
		group.Fields = groupFields
		return append(out, group)
	}
}

func generateField(out []*FieldYml, field Field, index, count int, noDoc bool) []*FieldYml {
	if index+1 == count {
		return insertLastField(out, field.Elements[index], field.Type, noDoc)
	}
	return insertGroup(out, field, index, count, noDoc)
}

func generateFields(f []Field, noDoc bool) []*FieldYml {
	out := make([]*FieldYml, 0)
	for _, ff := range f {
		out = generateField(out, ff, 1, len(ff.Elements), noDoc)
	}
	return out
}

func (p *Pipeline) toFieldsYml(noDoc bool) ([]byte, error) {
	pt, err := getPatternsFromProcessors(p.Processors)
	if err != nil {
		return nil, err
	}
	var fs []Field
	fs, err = getElementsFromPatterns(pt)
	if err != nil {
		return nil, err
	}

	f := generateFields(fs, noDoc)
	var d []byte
	d, err = yaml.Marshal(&f)

	return d, nil
}

func writeFieldsYml(module, fileset string, f []byte) error {
	p := fmt.Sprintf(fieldsYmlPath, module, fileset)
	err := ioutil.WriteFile(p, f, 0664)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	module := flag.String("module", "", "Name of the module to generate fields.yml for")
	fileset := flag.String("fileset", "", "Name of the fileset to generate fields.yml for")
	noDoc := flag.Bool("nodoc", false, "Generate description and example elements for fields.yml. Documentation is required, if the module is going to be submitted to elastic/beats.")
	flag.Parse()

	if *module == "" {
		log.Fatalln("Missing parameter: module")
	}
	if *fileset == "" {
		log.Fatalln("Missing parameter: fileset")
	}

	p, err := readPipeline(*module, *fileset)
	if err != nil {
		log.Fatalln("Error opening pipeline file: %v", err)
	}

	var d []byte
	d, err = p.toFieldsYml(*noDoc)
	if err != nil {
		log.Fatalln("Error while creating fields struct: %v", err)
	}

	err = writeFieldsYml(*module, *fileset, d)
	if err != nil {
		log.Fatalln("Error while writing fields.yml: %v", err)
	}
}
