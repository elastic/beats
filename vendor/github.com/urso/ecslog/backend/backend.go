package backend

import (
	"github.com/urso/ecslog/ctxtree"
)

type Backend interface {
	For(name string) Backend

	IsEnabled(lvl Level) bool
	UseContext() bool

	Log(Message)
}

type Level uint8

type Message struct {
	Name    string
	Level   Level
	Caller  Caller
	Message string
	Context ctxtree.Ctx
	Causes  []error
}

const (
	Trace Level = iota
	Debug
	Info
	Error
)

func (l Level) String() string {
	switch l {
	case Trace:
		return "trace"
	case Debug:
		return "debug"
	case Info:
		return "info"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}
