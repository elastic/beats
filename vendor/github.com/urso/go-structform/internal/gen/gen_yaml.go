package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/cfgutil"
	"github.com/elastic/go-ucfg/yaml"

	"golang.org/x/tools/imports"
)

var cfgOpts = []ucfg.Option{
	ucfg.PathSep("."),
	ucfg.ResolveEnv,
}

var datOpts = append([]ucfg.Option{ucfg.VarExp}, cfgOpts...)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	to := flag.String("o", "", "write to")
	format := flag.Bool("f", false, "format output using goimports")
	dataFile := flag.String("d", "", "input data file for use to fill out")

	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		return errors.New("Missing input file")
	}

	userData, err := loadData(*dataFile)
	if err != nil {
		return fmt.Errorf("Failed to read data file: %v", err)
	}

	gen := struct {
		Import    []string
		Templates map[string]string
		Main      string
	}{}
	if err = loadConfigInto(args[0], &gen); err != nil {
		errPrint("Failed to load script template")
		return err
	}

	dat := struct {
		Data *ucfg.Config
	}{}
	if err = loadConfigInto(args[0], &dat, ucfg.VarExp); err != nil {
		errPrint("Failed to load script data")
		return err
	}

	var T *template.Template
	D := cfgutil.NewCollector(nil, datOpts...)
	var data map[string]interface{}

	var defaultFuncs = template.FuncMap{
		"data":       func() map[string]interface{} { return data },
		"toLower":    strings.ToLower,
		"toUpper":    strings.ToUpper,
		"capitalize": strings.Title,
		"isnil": func(v interface{}) bool {
			return v == nil
		},
		"default": func(D, v interface{}) interface{} {
			if v == nil {
				return D
			}
			return v
		},
		"dict":   makeDict,
		"invoke": makeInvokeCommand(&T), // invoke another template with named parameters
	}

	var td *ucfg.Config
	T, td, err = loadTemplates(template.New("").Funcs(defaultFuncs), gen.Import)
	if err := D.Add(td, err); err != nil {
		errPrint("Failed to load imported template files")
		return err
	}

	if err := D.Add(dat.Data, nil); err != nil {
		errPrint("Failed to merge template data with top-level script")
		return err
	}

	if err := D.Add(ucfg.NewFrom(userData, datOpts...)); err != nil {
		errPrintf("Failed to merge user data")
		return err
	}

	if err := D.Config().Unpack(&data, datOpts...); err != nil {
		errPrint("Failed to unpack data")
		return err
	}

	T, err = addTemplates(T, gen.Templates)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	header := fmt.Sprintf("// This file has been generated from '%v', do not edit\n", args[0])
	buf.WriteString(header)
	T = T.New("master")
	T, err = T.Parse(gen.Main)
	if err != nil {
		return fmt.Errorf("Parsing 'template' fields failed with %v", err)
	}

	if err := T.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing template failed with %v", err)
	}

	content := buf.Bytes()
	if *format {
		content, err = imports.Process(*to, content, nil)
		if err != nil {
			return fmt.Errorf("Applying goimports failed with: %v", err)
		}
	}

	if *to != "" {
		return ioutil.WriteFile(*to, content, 0644)
	}

	_, err = os.Stdout.Write(content)
	return err
}

func loadTemplates(T *template.Template, files []string) (*template.Template, *ucfg.Config, error) {

	/*
		var childData []*ucfg.Config
		var templatesData []*ucfg.Config
	*/

	childData := cfgutil.NewCollector(nil, datOpts...)
	templateData := cfgutil.NewCollector(nil, datOpts...)

	for _, file := range files {
		gen := struct {
			Import    []string
			Templates map[string]string
		}{}

		dat := struct {
			Data *ucfg.Config
		}{}

		err := loadConfigInto(file, &gen)
		if err != nil {
			return nil, nil, err
		}

		var D *ucfg.Config
		T, D, err = loadTemplates(T, gen.Import)
		if err != nil {
			return nil, nil, err
		}

		T, err = addTemplates(T, gen.Templates)
		if err != nil {
			return nil, nil, err
		}

		err = loadConfigInto(file, &dat, ucfg.VarExp)
		if err != nil {
			errPrint("Failed to load data from: ", file)
			return nil, nil, err
		}

		childData.Add(D, nil)
		templateData.Add(dat.Data, nil)
	}

	if err := childData.Error(); err != nil {
		errPrintf("Procesing file %v: failed to merge child data: %v", files, err)
		return nil, nil, err
	}

	if err := templateData.Error(); err != nil {
		errPrintf("Procesing file %v: failed to merge template data: %v", files, err)
		return nil, nil, err
	}

	if err := childData.Add(templateData.Config(), templateData.Error()); err != nil {
		errPrintf("Failed to combine template data: ", err)
		return nil, nil, err
	}

	return T, childData.Config(), nil
}

func addTemplates(T *template.Template, templates map[string]string) (*template.Template, error) {
	for name, content := range templates {
		var err error

		T = T.New(name)
		T, err = T.Parse(content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %v: %v", name, err)
		}
	}

	return T, nil
}

func loadConfig(file string, extraOpts ...ucfg.Option) (cfg *ucfg.Config, err error) {
	opts := append(append([]ucfg.Option{}, extraOpts...), cfgOpts...)
	cfg, err = yaml.NewConfigWithFile(file, opts...)
	if err != nil {
		err = fmt.Errorf("Failed to read file %v with: %v", file, err)
	}
	return
}

func loadConfigInto(file string, to interface{}, extraOpts ...ucfg.Option) error {
	cfg, err := loadConfig(file, extraOpts...)
	if err == nil {
		err = readConfig(cfg, to, extraOpts...)
	}
	return err
}

func readConfig(cfg *ucfg.Config, to interface{}, extraOpts ...ucfg.Option) error {
	opts := append(append([]ucfg.Option{}, extraOpts...), cfgOpts...)
	if err := cfg.Unpack(to, opts...); err != nil {
		return fmt.Errorf("Parsing template file failed with %v", err)
	}
	return nil
}

func makeDict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}

	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

func makeInvokeCommand(T **template.Template) func(string, ...interface{}) (string, error) {
	return func(name string, values ...interface{}) (string, error) {
		params, err := makeDict(values...)
		if err != nil {
			return "", err
		}

		var buf bytes.Buffer
		err = (*T).ExecuteTemplate(&buf, name, params)
		return buf.String(), err

	}
}

func loadData(file string) (map[string]interface{}, error) {
	if file == "" {
		return nil, nil
	}

	meta := struct {
		Entries map[string]struct {
			Default     string
			Description string
		} `config:",inline"`
	}{}

	err := loadConfigInto(file, &meta, ucfg.VarExp)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(os.Stdin)

	state := map[string]interface{}{}
	for name, entry := range meta.Entries {
		// parse default value
		T, err := template.New("").Parse(entry.Default)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse data entry %v: %v", name, err)
		}

		var buf bytes.Buffer
		if err := T.Execute(&buf, state); err != nil {
			return nil, fmt.Errorf("Failed to evaluate data entry %v: %v", name, err)
		}

		// ask user for input
		defaultValue := buf.String()
		fmt.Printf("%v\n%v [%v]: ", entry.Description, name, defaultValue)
		value, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("Error waiting for user input: %v", err)
		}

		value = strings.TrimSpace(value)
		if value == "" {
			value = defaultValue
		}

		state[name] = value
	}

	return state, nil
}

func errPrint(msg ...interface{}) {
	fmt.Fprintln(os.Stderr, msg...)
}

func errPrintf(format string, msg ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", msg...)
}
