package ucfg

import "os"

type Option func(*options)

type options struct {
	tag          string
	validatorTag string
	pathSep      string
	meta         *Meta
	env          []*Config
	resolvers    []func(name string) (string, error)
	varexp       bool
}

func StructTag(tag string) Option {
	return func(o *options) {
		o.tag = tag
	}
}

func ValidatorTag(tag string) Option {
	return func(o *options) {
		o.validatorTag = tag
	}
}

func PathSep(sep string) Option {
	return func(o *options) {
		o.pathSep = sep
	}
}

func MetaData(meta Meta) Option {
	return func(o *options) {
		o.meta = &meta
	}
}

func Env(e *Config) Option {
	return func(o *options) {
		o.env = append(o.env, e)
	}
}

func Resolve(fn func(name string) (string, error)) Option {
	return func(o *options) {
		o.resolvers = append(o.resolvers, fn)
	}
}

var ResolveEnv = Resolve(func(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", ErrMissing
	}
	return value, nil
})

var VarExp Option = func(o *options) {
	o.varexp = true
}

func makeOptions(opts []Option) *options {
	o := options{
		tag:          "config",
		validatorTag: "validate",
		pathSep:      "", // no separator by default
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &o
}
