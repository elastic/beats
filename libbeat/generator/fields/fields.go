package fields

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	generatedFieldsYml = filepath.Join("_meta", "fields.generated.yml")
)

// YmlFile holds the info on files and how to write them into the global fields.yml
type YmlFile struct {
	Path   string
	Indent int
}

func collectBeatFiles(beatPath string, fieldFiles []*YmlFile) ([]*YmlFile, error) {
	commonFields := filepath.Join(beatPath, "_meta", "fields.common.yml")
	_, err := os.Stat(commonFields)
	if os.IsNotExist(err) {
		return fieldFiles, nil
	} else if err != nil {
		return nil, err
	}

	files := []*YmlFile{
		&YmlFile{
			Path:   commonFields,
			Indent: 0,
		},
	}

	return append(files, fieldFiles...), nil
}

func writeGeneratedFieldsYml(beatsPath string, fieldFiles []*YmlFile) error {
	outPath := path.Join(beatsPath, generatedFieldsYml)
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	fw := bufio.NewWriter(f)
	for _, p := range fieldFiles {
		ff, err := os.Open(p.Path)
		if err != nil {
			return err
		}
		defer ff.Close()

		fs := bufio.NewScanner(ff)
		for fs.Scan() {
			err = writeIndentedLine(fw, fs.Text()+"\n", p.Indent)
			if err != nil {
				return err
			}

		}
		if err := fs.Err(); err != nil {
			return err
		}
	}
	return nil
}

func writeIndentedLine(fw *bufio.Writer, l string, indent int) error {
	ll := strings.Repeat(" ", indent) + l
	fmt.Fprint(fw, ll)
	return fw.Flush()
}

// Generate collects fields.yml files and concatenates them into one global file.
func Generate(esBeatsPath, beatPath string, files []*YmlFile) error {
	files, err := collectBeatFiles(beatPath, files)
	if err != nil {
		return err
	}

	err = writeGeneratedFieldsYml(beatPath, files)
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
		return copyFileWithFlag(generatedPath, out, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
	}

	libbeatPath := filepath.Join(esBeatsPath, "libbeat", generatedFieldsYml)
	out := filepath.Join(beatPath, "fields.yml")
	err = copyFileWithFlag(libbeatPath, out, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return err
	}
	return copyFileWithFlag(generatedPath, out, os.O_WRONLY|os.O_APPEND)
}

func isLibbeat(beatPath string) bool {
	return filepath.Base(beatPath) == "libbeat"
}

func createIfNotExists(inPath, outPath string) error {
	_, err := os.Stat(outPath)
	if os.IsNotExist(err) {
		err := copyFileWithFlag(inPath, outPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
		if err != nil {
			fmt.Println("Cannot find _meta/fields.yml")
		}
		return nil
	}
	return err
}

func copyFileWithFlag(in, out string, flag int) error {
	input, err := ioutil.ReadFile(in)
	if err != nil {
		return err
	}

	output, err := os.OpenFile(out, flag, 0644)
	if err != nil {
		return err
	}
	defer output.Close()

	_, err = output.Write(input)
	return err

}
