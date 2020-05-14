package gotool

// Args holds parameters, environment variables and flag information used to
// pass to the go tool.
type Args struct {
	// Extra flags one can pass to a go command wrapper.
	Extra map[string]string

	// Environment variables to set when calling a go command.
	Environment map[string]string

	// Flags sets the CLI flags to be passed
	Flags map[string]string

	// Positional configured positional arguments
	Positional []string
}

// ArgOpt is a functional option setting fields within Args once executed.
type ArgOpt func(args *Args)

// SetExtra sets a 'special' value
func (a *Args) SetExtra(k, v string) {
	if a.Extra == nil {
		a.Extra = map[string]string{}
	}
	a.Extra[k] = v
}

// SetEnv sets an environmant variable to be passed to the child process on exec.
func (a *Args) SetEnv(k, v string) {
	if a.Environment == nil {
		a.Environment = map[string]string{}
	}
	a.Environment[k] = v
}
