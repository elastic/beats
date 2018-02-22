package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

const defaultGlob = "module/*/_meta/config*.yml.tpl"

var (
	goos      = flag.String("os", runtime.GOOS, "generate config specific to the specified operating system")
	goarch    = flag.String("arch", runtime.GOARCH, "generate config specific to the specified CPU architecture")
	reference = flag.Bool("ref", false, "generate a reference config")
	concat    = flag.Bool("concat", false, "concatenate all configs instead writing individual files")
)

func findConfigFiles(globs []string) ([]string, error) {
	var configFiles []string
	for _, glob := range globs {
		files, err := filepath.Glob(glob)
		if err != nil {
			return nil, errors.Wrapf(err, "failed on glob %v", glob)
		}
		configFiles = append(configFiles, files...)
	}
	return configFiles, nil
}

func getConfig(file string) ([]byte, error) {
	tpl, err := template.ParseFiles(file)
	if err != nil {
		return nil, errors.Wrapf(err, "failed reading %v", file)
	}

	var archBits string
	switch *goarch {
	case "i386":
		archBits = "32"
	case "amd64":
		archBits = "64"
	default:
		return nil, fmt.Errorf("supporting only i386 and amd64 architecture")
	}
	data := map[string]interface{}{
		"goarch":    *goarch,
		"goos":      *goos,
		"reference": *reference,
		"arch_bits": archBits,
	}
	buf := new(bytes.Buffer)
	if err = tpl.Execute(buf, data); err != nil {
		return nil, errors.Wrapf(err, "failed executing template %v", file)
	}

	return buf.Bytes(), nil
}

func output(content []byte, file string) error {
	if file == "-" {
		fmt.Println(string(content))
		return nil
	}

	if err := ioutil.WriteFile(file, content, 0640); err != nil {
		return errors.Wrapf(err, "failed writing output to %v", file)
	}

	return nil
}

func logAndExit(err error) {
	fmt.Fprintf(os.Stderr, "%+v\n", err)
	os.Exit(1)
}

func main() {
	flag.Parse()

	globs := os.Args
	if len(os.Args) > 0 {
		path, err := filepath.Abs(defaultGlob)
		if err != nil {
			logAndExit(err)
		}
		globs = []string{path}
	}

	files, err := findConfigFiles(globs)
	if err != nil {
		logAndExit(err)
	}

	var segments [][]byte
	for _, file := range files {
		segment, err := getConfig(file)
		if err != nil {
			logAndExit(err)
		}

		if *concat {
			segments = append(segments, segment)
		} else {
			output(segment, strings.TrimSuffix(file, ".tpl"))
		}
	}

	if *concat {
		if err := output(bytes.Join(segments, []byte{'\n'}), "-"); err != nil {
			logAndExit(err)
		}
	}
}
