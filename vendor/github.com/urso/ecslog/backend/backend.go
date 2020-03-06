// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

package backend

import "github.com/urso/diag"

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
	Context *diag.Context
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
