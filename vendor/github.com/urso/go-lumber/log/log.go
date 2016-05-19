package log

import "log"

type Logging interface {
	Printf(string, ...interface{})
	Println(...interface{})
	Print(...interface{})
}

type defaultLogger struct{}

// The logger use by go-lumber
var Logger Logging = defaultLogger{}

func (defaultLogger) Printf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func (defaultLogger) Println(args ...interface{}) {
	log.Println(args...)
}

func (defaultLogger) Print(args ...interface{}) {
	log.Print(args...)
}

func Printf(format string, args ...interface{}) {
	Logger.Printf(format, args...)
}

func Println(args ...interface{}) {
	Logger.Println(args...)
}

func Print(args ...interface{}) {
	Logger.Print(args...)
}
