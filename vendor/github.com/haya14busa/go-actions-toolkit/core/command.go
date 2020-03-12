package core

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const cmdString = "::"

type CommandProperties map[string]string

// NewCommandProperties returns a new CommandProperties from a list of key,
// value string pairs.
//
// NewCommandProperties panics if given an odd number of arguments.
func NewCommandProperties(kv ...string) CommandProperties {
	if len(kv)%2 == 1 {
		panic("core.NewCommandProperties: odd argument count")
	}
	p := make(map[string]string)
	for i := 0; i < len(kv); i += 2 {
		p[kv[i]] = kv[i+1]
	}
	return p
}

// Add adds new properties.
func (cp CommandProperties) Add(key, value string) {
	cp[key] = value
}

// AddLine adds new line property.
func (cp CommandProperties) AddLine(value int) {
	cp.Add("line", strconv.Itoa(value))
}

// AddLine adds new col property.
func (cp CommandProperties) AddCol(value int) {
	cp.Add("col", strconv.Itoa(value))
}

// AddLine adds new file property.
func (cp CommandProperties) AddFile(value string) {
	cp.Add("file", value)
}

// IssueCommand issues a Logging command of GitHub Actions.
//
// https://help.github.com/en/actions/automating-your-workflow-with-github-actions/development-tools-for-github-actions#logging-commands
func IssueCommand(command string, properties CommandProperties, message string) {
	cmd := NewCommand(command, properties, message)
	fmt.Fprintln(os.Stdout, cmd.String())
}

// Issue issues a Logging command of GitHub Actions.
//
// https://help.github.com/en/actions/automating-your-workflow-with-github-actions/development-tools-for-github-actions#logging-commands
func Issue(command string, message string) {
	IssueCommand(command, nil, message)
}

// NewCommand creates a Logging command of GitHub Actions.
//
// https://help.github.com/en/actions/automating-your-workflow-with-github-actions/development-tools-for-github-actions#logging-commands
func NewCommand(command string, properties CommandProperties, message string) Command {
	return Command{Command: command, Properties: properties, Message: message}
}

type Command struct {
	Command    string
	Properties CommandProperties
	Message    string
}

// String returns command in logging command format.
//
// It's port of https://github.com/actions/toolkit/blob/bfd29dcef82f324e9fb8855b6083fc0c2902bc27/packages/core/src/core.ts
func (c Command) String() string {
	var cmdStr strings.Builder
	cmdStr.WriteString(cmdString + c.Command)

	first := true
	for key, val := range c.Properties {
		if first {
			first = false
			cmdStr.WriteString(" ")
		} else {
			cmdStr.WriteString(",")
		}

		cmdStr.WriteString(fmt.Sprintf("%s=%s", key, escape(val)))
	}

	cmdStr.WriteString(cmdString)
	cmdStr.WriteString(escapeData(c.Message))
	return cmdStr.String()
}

func escapeData(s string) string {
	return strings.NewReplacer("\r", "%0D", "\n", "%0A").Replace(s)
}

func escape(s string) string {
	return strings.NewReplacer(
		"\r", "%0D",
		"\n", "%0A",
		"]", "%5D",
		";", "%3B",
	).Replace(s)
}
