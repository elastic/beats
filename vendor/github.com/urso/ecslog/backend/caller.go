// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

package backend

import "runtime"

type Caller struct {
	PC       uintptr
	file     string
	function string
	line     int
}

func GetCaller(skip int) Caller {
	var tmp [1]uintptr
	runtime.Callers(skip+2, tmp[:])
	return Caller{PC: tmp[0]}
}

func (c *Caller) File() string {
	if c.PC == 0 || c.file != "" {
		return c.file
	}
	c.load()
	return c.file
}

func (c *Caller) Function() string {
	if c.PC == 0 || c.function != "" {
		return c.function
	}
	c.load()
	return c.function
}

func (c *Caller) Line() int {
	if c.PC == 0 || c.file != "" {
		return c.line
	}
	c.load()
	return c.line
}

func (c *Caller) load() {
	fn := runtime.FuncForPC(c.PC - 1)
	if fn != nil {
		f, l := fn.FileLine(c.PC - 1)
		c.file = f
		c.line = l
		c.function = fn.Name()
	}
}
