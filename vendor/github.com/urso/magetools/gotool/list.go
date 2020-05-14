package gotool

import (
	"context"
	"os"
	"strings"

	"github.com/urso/magetools/clitool"
)

type GoList func(ctx context.Context, opts ...clitool.ArgOpt) ([]string, error)

type goList struct {
	g *Go
}

func makeList(g *Go) GoList {
	gl := &goList{g}
	return gl.Do
}

func (gl *goList) Do(ctx context.Context, opts ...clitool.ArgOpt) ([]string, error) {
	var buf strings.Builder
	err := gl.g.ExecGo(ctx, []string{"list"}, clitool.CreateArgs(opts...), &buf, os.Stderr)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(buf.String(), "\n")
	res := lines[:0]
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			res = append(res, line)
		}
	}

	return res, nil
}

func (fn GoList) ProjectPackages() ([]string, error) {
	return fn.Packages("./...")
}

func (fn GoList) Packages(pkgs ...string) ([]string, error) {
	return fn(context.Background(), clitool.Positional(pkgs...))
}

func (fn GoList) TestFiles(pkg string) ([]string, error) {
	const tmpl = `{{ range .TestGoFiles }}{{ printf "%s\n" . }}{{ end }}` +
		`{{ range .XTestGoFiles }}{{ printf "%s\n" . }}{{ end }}`

	return fn(context.Background(),
		clitool.Flag("-f", tmpl),
		clitool.Positional(pkg))
}

func (fn GoList) HasTests(pkg string) (bool, error) {
	files, err := fn.TestFiles(pkg)
	if err != nil {
		return false, err
	}
	return len(files) > 0, nil
}
