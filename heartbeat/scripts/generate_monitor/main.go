package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
)

func main() {
	var (
		monitor       string
		generatorHome string
		path          string
	)
	flag.StringVar(&monitor, "monitor", "", "Monitor name")
	flag.StringVar(&generatorHome, "home", "./scripts/generator/{{monitor}}", "Generator home path")
	flag.StringVar(&path, "path", "./monitors/active", "monitor output directory")
	flag.Parse()

	if monitor == "" {
		if err := prompt("Monitor name [example]: ", &monitor); err != nil {
			fatal(err)
		}

		if monitor == "" {
			monitor = "example"
		}
	}

	env := map[string]interface{}{
		// variables
		"monitor": noDot(monitor),

		// functions
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
	}
	if err := generate(generatorHome, filepath.Join(path, monitor), env); err != nil {
		fatal(err)
	}
}

func prompt(msg string, to interface{}) error {
	fmt.Print(msg)
	_, err := fmt.Scanln(to)
	return err
}

// create a template function, such that the variable can be accessed without
// the leading `.`.
func noDot(v string) interface{} {
	return func() string { return v }
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func generate(generatorPath string, outPath string, env map[string]interface{}) error {
	root, err := filepath.Abs(generatorPath)
	if err != nil {
		return err
	}

	outPath, err = filepath.Abs(outPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outPath, os.ModeDir|os.ModePerm); err != nil {
		return err
	}

	const tmplExt = ".tmpl"

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}

		name, err := execNameTemplate(path[len(root)+1:], env)
		if err != nil {
			return err
		}

		if info.IsDir() {
			dir := filepath.Join(outPath, name)
			return os.MkdirAll(dir, os.ModeDir|os.ModePerm)
		}

		if filepath.Ext(name) != tmplExt {
			return copyFile(filepath.Join(outPath, name), path)
		}

		// template file.
		name = name[:len(name)-len(tmplExt)]
		return copyTemplate(filepath.Join(outPath, name), path, env)
	})
}

func execNameTemplate(in string, env map[string]interface{}) (string, error) {
	var err error
	tmpl := withFunc(template.New("name"), env)
	tmpl, err = tmpl.Parse(in)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, withEnv(env))
	return buf.String(), err
}

func copyTemplate(to, tmplPath string, env map[string]interface{}) error {
	var err error
	tmpl := withFunc(template.New(filepath.Base(tmplPath)), env)
	tmpl, err = tmpl.ParseFiles(tmplPath)
	if err != nil {
		return err
	}

	out, err := os.Create(to)
	if err != nil {
		return err
	}
	defer out.Close()

	tmpl = tmpl.Funcs(env)
	return tmpl.Execute(out, env)
}

func copyFile(to, from string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(to)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func withFunc(tmpl *template.Template, env map[string]interface{}) *template.Template {
	return tmpl.Funcs(filterEnv(func(v interface{}) bool {
		return reflect.TypeOf(v).Kind() == reflect.Func
	}, env))
}

func withEnv(env map[string]interface{}) map[string]interface{} {
	return filterEnv(func(v interface{}) bool {
		return reflect.TypeOf(v).Kind() != reflect.Func
	}, env)
}

func filterEnv(pred func(interface{}) bool, env map[string]interface{}) map[string]interface{} {
	tmp := map[string]interface{}{}
	for k, v := range env {
		if pred(v) {
			tmp[k] = v
		}
	}

	if len(tmp) == 0 {
		return nil
	}
	return tmp
}
