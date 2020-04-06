package clitool

// ArgOpt is a functional option setting fields within Args once executed.
type ArgOpt func(args *Args)

func CreateArgs(opts ...ArgOpt) *Args {
	a := &Args{}
	Combine(opts...)(a)
	return a
}

func Positional(values ...string) ArgOpt {
	return func(a *Args) {
		for _, value := range values {
			if value != "" {
				a.Add(value)
			}
		}
	}
}

func Combine(opts ...ArgOpt) ArgOpt {
	return func(a *Args) {
		for _, opt := range opts {
			if opts != nil {
				opt(a)
			}
		}
	}
}

func Extra(key, value string) ArgOpt {
	return func(a *Args) { a.SetExtra(key, value) }
}

func ExtraIf(key, value string) ArgOpt {
	return SetIf(Extra, key, value)
}

func Env(key, value string) ArgOpt {
	return func(a *Args) { a.SetEnv(key, value) }
}

func EnvIf(key, value string) ArgOpt {
	return SetIf(Env, key, value)
}

func Flag(key, value string) ArgOpt {
	return func(a *Args) { a.SetFlag(key, value) }
}

func BoolFlag(key string, b bool) ArgOpt {
	if !b {
		return func(a *Args) {}
	}
	return Flag(key, "")
}

func FlagIf(key, value string) ArgOpt {
	return SetIf(Flag, key, value)
}

func SetIf(fn func(k, v string) ArgOpt, key, value string) ArgOpt {
	if value == "" {
		return Noop()
	}

	return fn(key, value)
}

func Noop() ArgOpt {
	return func(a *Args) {}
}

func When(b bool, opt ArgOpt) ArgOpt {
	if b {
		return opt
	}
	return Noop()
}
