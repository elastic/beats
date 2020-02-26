package clitool

import (
	"context"
	"io"
)

type Command struct {
	Path       string
	SubCommand []string
	WorkingDir string
}

func (c *Command) ExecCollectOutput(
	ctx context.Context,
	args *Args,
) (string, error) {
	return NewCLIExecutor(false).ExecCollectOutput(ctx, *c, args)
}

func (c *Command) Exec(
	ctx context.Context,
	args *Args,
	stdout, stderr io.Writer,
) (bool, error) {
	return NewCLIExecutor(false).Exec(ctx, *c, args, stdout, stderr)
}
