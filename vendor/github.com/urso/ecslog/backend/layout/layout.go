package layout

import (
	"io"

	"github.com/urso/ecslog/backend"
)

type Factory func(io.Writer) (Layout, error)

type Layout interface {
	UseContext() bool
	Log(msg backend.Message)
}
