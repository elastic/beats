// Package gotool provides some common wrappers around the go tool, for
// executing go commands within go based build scripts.
package gotool

import (
	"context"
	"io"

	"github.com/urso/magetools/clitool"
)

type Go struct {
	Path       string
	WorkingDir string
	Executor   clitool.Executor

	Build GoBuild
	List  GoList
	Test  GoTest
}

func New(exec clitool.Executor, path string) *Go {
	if path == "" {
		path = "go"
	}
	g := &Go{Executor: exec, Path: path}
	g.Build = makeBuild(g)
	g.List = makeList(g)
	g.Test = makeTest(g)
	return g
}

func (g *Go) ExecGo(
	context context.Context,
	subCommands []string,
	args *clitool.Args,
	stdout, stderr io.Writer,
) error {
	cmd := clitool.Command{
		Path:       g.Path,
		SubCommand: subCommands,
		WorkingDir: g.WorkingDir,
	}

	execer := g.Executor
	if execer == nil {
		execer = clitool.NewCLIExecutor(false)
	}

	_, err := execer.Exec(context, cmd, args, stdout, stderr)
	return err
}

func (g *Go) Exec(
	context context.Context,
	path string,
	args *clitool.Args,
	stdout, stderr io.Writer,
) error {
	cmd := clitool.Command{
		Path:       path,
		WorkingDir: g.WorkingDir,
	}

	execer := g.Executor
	if execer == nil {
		execer = clitool.NewCLIExecutor(false)
	}

	_, err := execer.Exec(context, cmd, args, stdout, stderr)
	return err
}
