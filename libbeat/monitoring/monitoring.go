package monitoring

import "errors"

type Mode uint8

//go:generate stringer -type=Mode
const (
	// Reported mode, is lowest report level with most basic metrics only
	Reported Mode = iota

	// Full reports all metrics
	Full
)

// Default is the global default metrics registry provided by the monitoring package.
var Default = NewRegistry()

var errNotFound = errors.New("Name unknown")
var errInvalidName = errors.New("Name does not point to a valid variable")

func VisitMode(mode Mode, vs Visitor) {
	Default.Visit(mode, vs)
}

func Visit(vs Visitor) {
	Default.Visit(Full, vs)
}

func Do(mode Mode, f func(string, interface{})) {
	Default.Do(mode, f)
}

func Get(name string) Var {
	return Default.Get(name)
}

func GetRegistry(name string) *Registry {
	return Default.GetRegistry(name)
}

func Remove(name string) {
	Default.Remove(name)
}

func Clear() error {
	return Default.Clear()
}
