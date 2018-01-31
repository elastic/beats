package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

func copyTemplatesToDest(templatesPath, name, filesetPath, module, fileset string) error {
	template := path.Join(templatesPath, name)
	dest := path.Join(filesetPath, name)
	return copyTemplate(template, dest, module, fileset)
}

func readTemplate(template, module, fileset string) ([]byte, error) {
	c, err := ioutil.ReadFile(template)
	if err != nil {
		return []byte{}, fmt.Errorf("cannot read template: %v", err)
	}

	c = bytes.Replace(c, []byte("{module}"), []byte(module), -1)
	c = bytes.Replace(c, []byte("{fileset}"), []byte(fileset), -1)

	return c, nil
}

func copyTemplate(template, dest, module, fileset string) error {
	c, err := readTemplate(template, module, fileset)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dest, c, os.ModePerm)
	if err != nil {
		return fmt.Errorf("cannot copy template: %v", err)
	}
	return nil
}

func appendTemplate(template, dest, module, fileset string) error {
	c, err := readTemplate(template, module, fileset)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err == nil {
		_, err = f.Write(c)
	}
	if err != nil {
		return fmt.Errorf("cannot append template: %v", err)
	}

	return nil
}

func generateModule(module, fileset, modulePath, beatsPath string) error {
	p := path.Join(modulePath, "module", module)
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		return nil
	}

	d := path.Join(p, "_meta", "kibana", "default")
	err := os.MkdirAll(d, 0750)
	if err != nil {
		return err
	}

	templatesPath := path.Join(beatsPath, "scripts", "module")
	filesToCopy := []string{path.Join("_meta", "fields.yml"), path.Join("_meta", "docs.asciidoc"), path.Join("module.yml")}
	for _, f := range filesToCopy {
		err := copyTemplatesToDest(templatesPath, f, p, module, fileset)
		if err != nil {
			return err
		}
	}

	return nil
}

func generateFileset(module, fileset, modulePath, beatsPath string) error {
	filesetPath := path.Join(modulePath, "module", module, fileset)
	if _, err := os.Stat(filesetPath); !os.IsNotExist(err) {
		return fmt.Errorf("fileset already exists: %s", fileset)
	}

	dirsToCreate := []string{"", "_meta", "test", "config", "ingest"}
	for _, d := range dirsToCreate {
		p := path.Join(filesetPath, d)
		err := os.Mkdir(p, 0750)
		if err != nil {
			return err
		}
	}

	templatesPath := path.Join(beatsPath, "scripts", "module", "fileset")
	filesToCopy := []string{path.Join("config", "config.yml"), path.Join("ingest", "pipeline.json"), "manifest.yml"}
	for _, f := range filesToCopy {
		err := copyTemplatesToDest(templatesPath, f, filesetPath, module, fileset)
		if err != nil {
			return err
		}
	}

	return addFilesetDashboard(module, fileset, modulePath, beatsPath)
}

func addFilesetDashboard(module, fileset, modulePath, beatsPath string) error {
	templatesPath := path.Join(beatsPath, "scripts", "module")
	template := path.Join(templatesPath, "module-fileset.yml")
	dest := path.Join(modulePath, "module", module, "module.yml")
	return appendTemplate(template, dest, module, fileset)
}

func main() {
	module := flag.String("module", "", "Name of the module")
	fileset := flag.String("fileset", "", "Name of the fileset")
	modulePath := flag.String("path", ".", "Path to the generated fileset")
	beatsPath := flag.String("beats_path", ".", "Path to elastic/beats")
	flag.Parse()

	if *module == "" {
		fmt.Println("Missing parameter: module")
		os.Exit(1)
	}

	if *fileset == "" {
		fmt.Println("Missing parameter: fileset")
		os.Exit(1)
	}

	err := generateModule(*module, *fileset, *modulePath, *beatsPath)
	if err != nil {
		fmt.Printf("Cannot generate module: %v\n", err)
		os.Exit(2)
	}

	err = generateFileset(*module, *fileset, *modulePath, *beatsPath)
	if err != nil {
		fmt.Printf("Cannot generate fileset: %v\n", err)
		os.Exit(3)
	}

	fmt.Println("New module was generated, please check that module.yml file have proper fileset dashboard settings. After setting up Grok pattern in pipeline.json, please generate fields.yml")
}
