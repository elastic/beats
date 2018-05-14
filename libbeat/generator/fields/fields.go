package fields

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

var (
	generatedFieldsYml = filepath.Join("_meta", "fields.generated.yml")
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

	return append(files, fieldFiles...)
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

	return bytes.Join([][]byte{newline, content, newline}, empty)
}

// Generate collects fields.yml files and concatenates them into one global file.
func Generate(beatsPath, beatName string, files []*YmlFile) error {
	files = collectBeatFiles(beatsPath, beatName, files)

	err := os.MkdirAll(filepath.Join(beatsPath, beatName, "_meta"), 0644)
	if err != nil {
		return err
	}

	return writeGeneratedFieldsYml(beatsPath, beatName, files)
}
