package monitoring

import "errors"

// Default is the global default metrics registry provided by the monitoring package.
var Default = NewRegistry()

var errNotFound = errors.New("Name unknown")
var errInvalidName = errors.New("Name does not point to a valid variable")

func Visit(vs Visitor) error {
	return Default.Visit(vs)
}

func Do(f func(string, interface{}) error) error {
	return Default.Do(f)
}

func Get(name string) interface{} {
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
