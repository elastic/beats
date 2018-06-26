package feature

//go:generate stringer -type=Stability

// Stability defines the stability of the feature, this value can be used to filter a bundler.
type Stability int

// List all the available stability for a feature.
const (
	Undefined Stability = iota
	Stable
	Beta
	Experimental
)
