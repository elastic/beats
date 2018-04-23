package fields

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

const (
	generatedFieldsYml = "_meta/fields.generated.yml"
	commonFieldsYml    = "libbeat/_meta/fields.common.yml"
	libbeatFields      = "libbeat/processors/*/_meta/fields.yml"
)

type YmlFile struct {
	Path   string
	Indent int
}

func collectBeatFiles(beatsPath, name string, fieldFiles []*YmlFile) []*YmlFile {
	files := []*YmlFile{
		&YmlFile{
			Path:   filepath.Join(beatsPath, name, "_meta", "fields.common.yml"),
			Indent: 0,
		},
	}

	//processors := collectProcessorsFields(beatsPath)
	//files = append(files, processors...)
	return append(files, fieldFiles...)
}

func collectProcessorsFields(beatsPath string) []string {
	p := filepath.Join(beatsPath, "libbeat", "processors")
	processors, err := ioutil.ReadDir(p)

	var names []string
	for _, pp := range processors {
		if !pp.IsDir() {
			continue
		}
		fieldsYmlPath := filepath.Join(p, pp.Name(), "_meta", "fields.yml")
		if _, err = os.Stat(fieldsYmlPath); !os.IsNotExist(err) {
			names = append(names, fieldsYmlPath)
		}
	}
	return names
}

func writeGeneratedFieldsYml(beatsPath, name string, fieldFiles []*YmlFile) error {
	outPath := path.Join(beatsPath, name, generatedFieldsYml)
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, p := range fieldFiles {
		content, err := ioutil.ReadFile(p.Path)
		if err != nil {
			return err
		}

		content = indent(content, p.Indent)

		_, err = f.Write(content)
		if err != nil {
			return err
		}
	}
	return nil
}

func indent(content []byte, n int) []byte {
	newline := []byte("\n")
	empty := []byte("")
	i := bytes.Repeat([]byte(" "), n)
	c := bytes.Join([][]byte{newline, i}, empty)

	content = bytes.Join([][]byte{i, content}, empty)
	content = bytes.TrimRight(content, "\n")
	content = bytes.Replace(content, newline, c, -1)
	content = bytes.TrimRight(content, " ")

	return bytes.Join([][]byte{newline, content}, empty)
}

func Generate(beatsPath, beatName string, files []*YmlFile) error {
	files = collectBeatFiles(beatsPath, beatName, files)

	err := os.MkdirAll(filepath.Join(beatsPath, beatName, "_meta"), 0644)
	if err != nil {
		return err
	}

	return writeGeneratedFieldsYml(beatsPath, beatName, files)
}
