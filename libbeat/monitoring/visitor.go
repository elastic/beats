package monitoring

// Visitor interface supports traversing a monitoring registry
type Visitor interface {
	ValueVisitor
	RegistryVisitor
}

type ValueVisitor interface {
	OnString(s string) error
	OnBool(b bool) error
	OnInt(i int64) error
	OnFloat(f float64) error
}

type RegistryVisitor interface {
	OnRegistryStart() error
	OnRegistryFinished() error
	OnKey(s string) error
	OnKeyNext() error
}
