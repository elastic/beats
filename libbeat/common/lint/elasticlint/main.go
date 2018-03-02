package main

import (
	"go/build"
	"fmt"
	"os"
	"path/filepath"
	"io/ioutil"
	"github.com/elastic/beats/libbeat/common/lint"
)

// Problems:
// 1. Linter or Formatter?
// 2. make lint -- all files or scope from change?


// Inspired by:
// https://github.com/golang/lint/blob/master/lint.go
// https://github.com/golang/example/tree/master/gotypes (CheckNilFuncComparison)
// https://github.com/golang/tools/commit/6d70fb2e85323e81c89374331d3d2b93304faa36 (tests)

func main() {
	lintDir(".")
}

func lintFiles(filenames ...string) {
	linters := []lint.Linter{&lint.ErrorCheckLinter{}}

	for _, filename := range filenames {
		src, err := ioutil.ReadFile(filename)

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		content := string(src)

		var problems []*lint.Problem

		for _, linter := range linters {
			problems, err = linter.Lint(content)
			if err != nil {
				panic(err)
			}
		}

		if len(problems) > 0 {
			for _, problem := range problems {
				fmt.Printf("%s:%v: %s\n", filename, problem.Position.Line, problem)
			}
		}
	}
}

func lintDir(dirname string) {
	pkg, err := build.ImportDir(dirname, 0)
	lintImportedPackage(pkg, err)
}

func lintImportedPackage(pkg *build.Package, err error) {
	if err != nil {
		if _, nogo := err.(*build.NoGoError); nogo {
			// Don't complain if the failure is due to no Go source files.
			return
		}
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var files []string
	files = append(files, pkg.GoFiles...)
	files = append(files, pkg.CgoFiles...)
	files = append(files, pkg.TestGoFiles...)
	if pkg.Dir != "." {
		for i, f := range files {
			files[i] = filepath.Join(pkg.Dir, f)
		}
	}

	lintFiles(files...)
}
