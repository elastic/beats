// Package log provides logging functionality used in go-lumber.
//
// The log package provides replaceable logging for use from within go-lumber.
// Overwrite Logging variable with custom Logging implementation for integrating
// go-lumber logging with applications logging strategy.
package log

import "log"

// Logging interface custom loggers must implement.
type Logging interface {
	Printf(string, ...interface{})
	Println(...interface{})
	Print(...interface{})
}

type defaultLogger struct{}

// Logger provides the global logger used by go-lumber.
var Logger Logging = defaultLogger{}

// Printf calls Logger.Printf to print to the standard logger. Arguments are
// handled in the manner of fmt.Printf.
func Printf(format string, args ...interface{}) {
	Logger.Printf(format, args...)
}

// Println calls Logger.Println to print to the standard logger. Arguments are
// handled in the manner of fmt.Println.
func Println(args ...interface{}) {
	Logger.Println(args...)
}

// Print calls Logger.Print to print to the standard logger. Arguments are
// handled in the manner of fmt.Print.
func Print(args ...interface{}) {
	Logger.Print(args...)
}

func (defaultLogger) Printf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func (defaultLogger) Println(args ...interface{}) {
	log.Println(args...)
}

func (defaultLogger) Print(args ...interface{}) {
	log.Print(args...)
}
