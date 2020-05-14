// Package core is Go port of @actions/core.
// Core functions for setting results, logging, registering secrets and
// exporting variables across actions.
//
// https://github.com/actions/toolkit/tree/master/packages/core
package core

import (
	"fmt"
	"os"
	"strings"
)

//-----------------------------------------------------------------------
// Variables
//-----------------------------------------------------------------------

// Sets env variable for this action and future actions in the job.
func ExportVariable(name, value string) {
	os.Setenv(name, value)
	IssueCommand("set-env", NewCommandProperties("name", name), value)
}

// Registers a secret which will get masked from logs.
func SetSecret(value string) {
	IssueCommand("add-mask", nil, value)
}

// Prepends inputPath to the PATH (for this action and future actions).
func AddPath(inputPath string) {
	IssueCommand("add-path", nil, inputPath)
	os.Setenv("PATH", inputPath+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// Gets the value of an input.  The value is also trimmed.
// It will return empty string if input is not present.
func GetInput(name string) string {
	val := os.Getenv(fmt.Sprintf("INPUT_%s",
		strings.ToUpper(strings.ReplaceAll(name, " ", "_"))))
	return strings.TrimSpace(val)
}

// Sets the value of an output.
func SetOutput(name, value string) {
	IssueCommand("set-output", NewCommandProperties("name", name), value)
}

//-----------------------------------------------------------------------
// Logging Commands
//-----------------------------------------------------------------------

// LogOption is logging option to indicate where message occurred.
type LogOption struct {
	File string
	Line int
	Col  int
}

func log(cmd string, mes string, opt *LogOption) {
	p := NewCommandProperties()
	if opt != nil {
		if opt.File != "" {
			p.AddFile(opt.File)
		}
		if opt.Line != 0 {
			p.AddLine(opt.Line)
		}
		if opt.Col != 0 {
			p.AddCol(opt.Col)
		}
	}
	IssueCommand(cmd, p, mes)
}

// Writes debug message to user log.
//
// You can optionally provide a filename (file), line number (line), and column
// (col) number as LogOption where the warning occurred.
func Debug(mes string, option *LogOption) { log("debug", mes, option) }

// Adds an error issue.
//
// You can optionally provide a filename (file), line number (line), and column
// (col) number as LogOption where the warning occurred.
func Error(mes string, option *LogOption) { log("error", mes, option) }

// Adds an warning issue.
//
// You can optionally provide a filename (file), line number (line), and column
// (col) number as LogOption where the warning occurred.
func Warning(mes string, option *LogOption) { log("warning", mes, option) }

// Writes info to log.
func Info(mes string) {
	fmt.Fprintln(os.Stdout, mes)
}

// Begin an output group.
// Output until the next `groupEnd` will be foldable in this group.
func StartGroup(name string) {
	Issue("group", name)
}

// End an output group.
func EndGroup() {
	Issue("endgroup", "")
}

//-----------------------------------------------------------------------
// Wrapper action state
//-----------------------------------------------------------------------

// Saves state for current action, the state can only be retrieved by this
// action's post job execution.
func SaveState(name, value string) {
	IssueCommand("save-state", NewCommandProperties("name", name), value)
}

// Gets the value of an state set by this action's main execution.
func GetState(name string) string {
	return os.Getenv(fmt.Sprintf("STATE_%s", name))
}
