package fields

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

var (
	generatedFieldsYml = filepath.Join("_meta", "fields.generated.yml")
)

// YmlFile holds the info on files and how to write them into the global fields.yml
type YmlFile struct {
	Path   string
	Indent int
}

func collectBeatFiles(beatPath string, fieldFiles []*YmlFile) []*YmlFile {
	commonFields := filepath.Join(beatPath, "_meta", "fields.common.yml")
	_, err := os.Stat(commonFields)
	if os.IsNotExist(err) {
		return fieldFiles
	}

	files := []*YmlFile{
		&YmlFile{
			Path:   commonFields,
			Indent: 0,
		},
	}

	return append(files, fieldFiles...)
}

func writeGeneratedFieldsYml(beatsPath string, fieldFiles []*YmlFile) error {
	outPath := path.Join(beatsPath, generatedFieldsYml)
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
func Generate(esBeatsPath, beatPath string, files []*YmlFile) error {
	files = collectBeatFiles(beatPath, files)

	err := writeGeneratedFieldsYml(beatPath, files)
	if err != nil {
		return err
	}

	return AppendFromLibbeat(esBeatsPath, beatPath)
}

// AppendFromLibbeat appends fields.yml of libbeat to the fields.yml
func AppendFromLibbeat(esBeatsPath, beatPath string) error {
	fieldsMetaPath := path.Join(beatPath, "_meta", "fields.yml")
	generatedPath := path.Join(beatPath, generatedFieldsYml)

	err := createIfNotExists(fieldsMetaPath, generatedPath)
	if err != nil {
		return err
	}

	if isLibbeat(beatPath) {
		out := filepath.Join(esBeatsPath, "libbeat", "fields.yml")
		return createFile(generatedPath, out)
	}

	libbeatPath := filepath.Join(esBeatsPath, "libbeat", generatedFieldsYml)
	out := filepath.Join(beatPath, "fields.yml")
	err = createFile(libbeatPath, out)
	if err != nil {
		return err
	}
	return appendGenerated(generatedPath, out)
}

func isLibbeat(beatPath string) bool {
	return filepath.Base(beatPath) == "libbeat"
}

func createIfNotExists(inPath, outPath string) error {
	_, err := os.Stat(outPath)
	if os.IsNotExist(err) {
		err := createFile(inPath, outPath)
		if err != nil {
			fmt.Println("Cannot find _meta/fields.yml")
		}
		return nil
	}
	return err
}

func createFile(in, out string) error {
	input, err := ioutil.ReadFile(in)
	if err != nil {
		return err
	}

	output, err := os.Create(out)
	if err != nil {
		return err
	}
	defer output.Close()

	_, err = output.Write(input)
	return err
}

func appendGenerated(generatedPath, fieldsPath string) error {
	input, err := ioutil.ReadFile(generatedPath)
	if err != nil {
		return err
	}

	output, err := os.OpenFile(fieldsPath, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer output.Close()

	_, err = output.Write(input)
	return err
}
