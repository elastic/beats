package clitool

import (
	"context"
	"io"
)

type Executor interface {
	Exec(
		ctx context.Context,
		cmd Command,
		args *Args,
		stdout, stderr io.Writer,
	) (bool, error)
}
