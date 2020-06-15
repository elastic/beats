package gotool

import (
	"context"
	"os"
	"strings"

	"github.com/urso/magetools/clitool"
)

type GoRun func(context context.Context, opts ...clitool.ArgOpt) error

type goRun struct {
	g *Go
}

func makeRun(g *Go) GoRun {
	gr := &goRun{g}
	return gr.Do
}

func (gr *goRun) Do(context context.Context, opts ...clitool.ArgOpt) error {
	return gr.g.ExecGo(context, []string{"run"}, clitool.CreateArgs(opts...), os.Stdout, os.Stderr)
}

func (GoRun) Tags(tags ...string) clitool.ArgOpt {
	return clitool.FlagIf("-tags", strings.Join(tags, " "))
}
func (GoRun) Script(files ...string) clitool.ArgOpt {
	return clitool.Positional(files...)
}
func (GoRun) ScriptArgs(opts ...clitool.ArgOpt) clitool.ArgOpt {
	args := clitool.CreateArgs(opts...).Build()
	return clitool.Positional(args...)
}
