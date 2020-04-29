package clitool

// Args holds parameters, environment variables and flag information used to
// pass to the go tool.
type Args struct {
	// Extra flags one can pass to a go command wrapper.
	Extra map[string]string

	// Environment variables to set when calling a go command.
	Environment map[string]string

	// Flags sets the CLI flags to be passed
	Flags []CommandFlag

	// Positional configured positional arguments
	Positional []string
}

type CommandFlag struct {
	Key   string
	Value string
}

func (a *Args) Build() []string {
	args := make([]string, 0, 2*len(a.Flags)+len(a.Positional))
	for _, f := range a.Flags {
		args = append(args, f.Key)
		if v := f.Value; v != "" {
			args = append(args, v)
		}
	}

	return append(args, a.Positional...)
}

// SetExtra sets a 'special' value
func (a *Args) SetExtra(k, v string) {
	if a.Extra == nil {
		a.Extra = map[string]string{}
	}
	a.Extra[k] = v
}

// GetExtra reads some value from 'extra' fields.
func (a *Args) GetExtra(k string) string {
	return a.GetExtraDefault(k, "")
}

// GetExtraDefault reads a value from 'extra' fields. Returns 'd' if the key is missing.
func (a *Args) GetExtraDefault(k, d string) string {
	if a.Extra == nil {
		return d
	}

	val, exists := a.Extra[k]
	if !exists {
		return d
	}

	return val
}

// SetEnv sets an environmant variable to be passed to the child process on exec.
func (a *Args) SetEnv(k, v string) {
	if a.Environment == nil {
		a.Environment = map[string]string{}
	}
	a.Environment[k] = v
}

// SetFlag sets an environment variable flag.
func (a *Args) SetFlag(flag, value string) {
	a.Flags = append(a.Flags, CommandFlag{flag, value})
}

// Add adds a positional argument to be passed to the child process on exec.
func (a *Args) Add(p string) {
	a.Positional = append(a.Positional, p)
}
