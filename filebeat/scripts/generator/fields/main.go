package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	pipelinePath  = "%s/module/%s/%s/ingest/pipeline.json"
	fieldsYmlPath = "%s/module/%s/%s/_meta/fields.yml"
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
		"LOCALDATETIME":   "text",
		"TIMESTAMP":       "text",
		"USERNAME":        "keyword",
		"WORD":            "keyword",
	}
)

type pipeline struct {
	Description string                   `json:"description"`
	Processors  []map[string]interface{} `json:"processors"`
	OnFailure   interface{}              `json:"on_failure"`
}

type field struct {
	Type     string
	Elements []string
}

type fieldYml struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description,omitempty"`
	Example     string      `yaml:"example,omitempty"`
	Type        string      `yaml:"type,omitempty"`
	Fields      []*fieldYml `yaml:"fields,omitempty"`
}

func newFieldYml(name, typeName string, noDoc bool) *fieldYml {
	if noDoc {
		return &fieldYml{
			Name: name,
			Type: typeName,
		}
	}

	return &fieldYml{
		Name:        name,
		Type:        typeName,
		Description: "Please add description",
		Example:     "Please add example",
	}
}

func newField(lp string) field {
	lp = lp[1 : len(lp)-1]
	ee := strings.Split(lp, ":")
	if len(ee) != 2 {
		return field{
			Type:     ee[0],
			Elements: nil,
		}
	}

	e := strings.Split(ee[1], ".")
	return field{
		Type:     ee[0],
		Elements: e,
	}
}

func readPipeline(beatsPath, module, fileset string) (*pipeline, error) {
	pp := fmt.Sprintf(pipelinePath, beatsPath, module, fileset)
	r, err := ioutil.ReadFile(pp)
	if err != nil {
		return nil, err
	}

	var p pipeline
	err = json.Unmarshal(r, &p)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func addNewField(fs []field, f field) []field {
	for _, ff := range fs {
		if reflect.DeepEqual(ff, f) {
			return fs
		}
	}
	return append(fs, f)
}

func getElementsFromPatterns(patterns []string) ([]field, error) {
	r, err := regexp.Compile("{[\\.\\w\\:]*}")
	if err != nil {
		return nil, err
	}

	var fs []field
	for _, lp := range patterns {
		pp := r.FindAllString(lp, -1)
		for _, p := range pp {
			f := newField(p)
			if f.Elements == nil {
				continue
			}
			fs = addNewField(fs, f)
		}

	}
	return fs, nil
}

func accumulatePatterns(grok interface{}) ([]string, error) {
	for k, v := range grok.(map[string]interface{}) {
		if k == "patterns" {
			vs := v.([]interface{})
			var p []string
			for _, s := range vs {
				p = append(p, s.(string))
			}
			return p, nil
		}
	}
	return nil, fmt.Errorf("No patterns in pipeline")
}

func accumulateRemoveFields(remove interface{}, out []string) []string {
	for k, v := range remove.(map[string]interface{}) {
		if k == "field" {
			vs := v.(string)
			return append(out, vs)
		}
	}
	return out
}

func accumulateRenameFields(rename interface{}, out map[string]string) map[string]string {
	var from, to string
	for k, v := range rename.(map[string]interface{}) {
		if k == "field" {
			from = v.(string)
		}
		if k == "target_field" {
			to = v.(string)
		}
	}
	out[from] = to
	return out
}

type processors struct {
	patterns []string
	remove   []string
	rename   map[string]string
}

func (p *processors) processFields() ([]field, error) {
	f, err := getElementsFromPatterns(p.patterns)
	if err != nil {
		return nil, err
	}

	for i, ff := range f {
		fs := strings.Join(ff.Elements, ".")
		for k, mv := range p.rename {
			if k == fs {
				ff.Elements = strings.Split(mv, ".")
			}
		}
		for _, rm := range p.remove {
			if fs == rm {
				f = append(f[:i], f[i+1:]...)
			}
		}
	}
	return f, nil
}

func getProcessors(p []map[string]interface{}) (*processors, error) {
	var patterns, rmFields []string
	mvFields := make(map[string]string)

	for _, e := range p {
		if ee, ok := e["grok"]; ok {
			pp, err := accumulatePatterns(ee)
			if err != nil {
				return nil, err
			}
			patterns = append(patterns, pp...)
		}
		if rm, ok := e["remove"]; ok {
			rmFields = accumulateRemoveFields(rm, rmFields)
		}
		if mv, ok := e["rename"]; ok {
			mvFields = accumulateRenameFields(mv, mvFields)
		}
	}

	if patterns == nil {
		return nil, fmt.Errorf("No patterns in pipeline")
	}

	return &processors{
		patterns: patterns,
		remove:   rmFields,
		rename:   mvFields,
	}, nil
}

func getFieldByName(f []*fieldYml, name string) *fieldYml {
	for _, ff := range f {
		if ff.Name == name {
			return ff
		}
	}
	return nil
}

func insertLastField(f []*fieldYml, name, typeName string, noDoc bool) []*fieldYml {
	ff := getFieldByName(f, name)
	if ff != nil {
		return f
	}

	nf := newFieldYml(name, types[typeName], noDoc)
	return append(f, nf)
}

func insertGroup(out []*fieldYml, field field, index, count int, noDoc bool) []*fieldYml {
	g := getFieldByName(out, field.Elements[index])
	if g != nil {
		g.Fields = generateField(g.Fields, field, index+1, count, noDoc)
		return out
	}

	var groupFields []*fieldYml
	groupFields = generateField(groupFields, field, index+1, count, noDoc)
	group := newFieldYml(field.Elements[index], "group", noDoc)
	group.Fields = groupFields
	return append(out, group)
}

func generateField(out []*fieldYml, field field, index, count int, noDoc bool) []*fieldYml {
	if index+1 == count {
		return insertLastField(out, field.Elements[index], field.Type, noDoc)
	}
	return insertGroup(out, field, index, count, noDoc)
}

func generateFields(f []field, noDoc bool) []*fieldYml {
	var out []*fieldYml
	for _, ff := range f {
		index := 1
		if len(ff.Elements) == 1 {
			index = 0
		}
		out = generateField(out, ff, index, len(ff.Elements), noDoc)
	}
	return out
}

func (p *pipeline) toFieldsYml(noDoc bool) ([]byte, error) {
	pr, err := getProcessors(p.Processors)
	if err != nil {
		return nil, err
	}

	var fs []field
	fs, err = pr.processFields()
	if err != nil {
		return nil, err
	}

	f := generateFields(fs, noDoc)
	var d []byte
	d, err = yaml.Marshal(&f)

	return d, nil
}

func writeFieldsYml(beatsPath, module, fileset string, f []byte) error {
	p := fmt.Sprintf(fieldsYmlPath, beatsPath, module, fileset)
	err := ioutil.WriteFile(p, f, 0664)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	module := flag.String("module", "", "Name of the module")
	fileset := flag.String("fileset", "", "Name of the fileset")
	beatsPath := flag.String("beats_path", ".", "Path to elastic/beats")
	noDoc := flag.Bool("nodoc", false, "Generate documentation for fields")
	flag.Parse()

	if *module == "" {
		fmt.Println("Missing parameter: module")
		os.Exit(1)
	}

	if *fileset == "" {
		fmt.Println("Missing parameter: fileset")
		os.Exit(1)
	}

	p, err := readPipeline(*beatsPath, *module, *fileset)
	if err != nil {
		fmt.Printf("Cannot read pipeline.yml of fileset: %v\n", err)
		os.Exit(2)
	}

	var d []byte
	d, err = p.toFieldsYml(*noDoc)
	if err != nil {
		fmt.Printf("Cannot generate fields.yml for fileset: %v\n", err)
		os.Exit(3)
	}

	err = writeFieldsYml(*beatsPath, *module, *fileset, d)
	if err != nil {
		fmt.Printf("Cannot write field.yml of fileset: %v\n", err)
		os.Exit(4)
	}

	fmt.Printf("Fields.yml generated for %s/%s\n", *module, *fileset)
}
