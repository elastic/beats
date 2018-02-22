package ucfg

import (
	"os"
)

// Option type implementing additional options to be passed
// to go-ucfg library functions.
type Option func(*options)

type options struct {
	tag          string
	validatorTag string
	pathSep      string
	meta         *Meta
	env          []*Config
	resolvers    []func(name string) (string, error)
	varexp       bool

	// temporary cache of parsed splice values for lifetime of call to
	// Unpack/Pack/Get/...
	parsed valueCache

	activeFields *fieldSet
}

type valueCache map[string]spliceValue

// id used to store intermediate parse results in current execution context.
// As parsing results might differ between multiple calls due to:
// splice being shared between multiple configurations, or environment
// changing between calls + lazy nature of cfgSplice, parsing results cannot
// be stored in cfgSplice itself.
type cacheID string

type spliceValue struct {
	err   error
	value value
}

// StructTag option sets the struct tag name to use for looking up
// field names and options in `Unpack` and `Merge`.
// The default struct tag in `config`.
func StructTag(tag string) Option {
	return func(o *options) {
		o.tag = tag
	}
}

// ValidatorTag option sets the struct tag name used to set validators
// on struct fields in `Unpack`.
// The default struct tag in `validate`.
func ValidatorTag(tag string) Option {
	return func(o *options) {
		o.validatorTag = tag
	}
}

// PathSep sets the path separator used to split up names into a tree like hierarchy.
// If PathSep is not set, field names will not be split.
func PathSep(sep string) Option {
	return func(o *options) {
		o.pathSep = sep
	}
}

// MetaData option passes additional metadata (currently only source of the
// configuration) to be stored internally (e.g. for error reporting).
func MetaData(meta Meta) Option {
	return func(o *options) {
		o.meta = &meta
	}
}

// Env option adds another configuration for variable expansion to be used, if
// the path to look up does not exist in the actual configuration. Env can be used
// multiple times in order to add more lookup environments.
func Env(e *Config) Option {
	return func(o *options) {
		o.env = append(o.env, e)
	}
}

// Resolve option adds a callback used by variable name expansion. The callback
// will be called if a variable can not be resolved from within the actual configuration
// or any of its environments.
func Resolve(fn func(name string) (string, error)) Option {
	return func(o *options) {
		o.resolvers = append(o.resolvers, fn)
	}
}

// ResolveEnv option adds a look up callback looking up values in the available
// OS environment variables.
var ResolveEnv Option = doResolveEnv

func doResolveEnv(o *options) {
	o.resolvers = append(o.resolvers, func(name string) (string, error) {
		value := os.Getenv(name)
		if value == "" {
			return "", ErrMissing
		}
		return value, nil
	})
}

// VarExp option enables support for variable expansion. Resolve and Env options will only be effective if  VarExp is set.
var VarExp Option = doVarExp

func doVarExp(o *options) { o.varexp = true }

func makeOptions(opts []Option) *options {
	o := options{
		tag:          "config",
		validatorTag: "validate",
		pathSep:      "", // no separator by default
		parsed:       map[string]spliceValue{},
		activeFields: NewFieldSet(nil),
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &o
}

func (cache valueCache) cachedValue(
	id cacheID,
	f func() (value, error),
) (value, error) {
	if v, ok := cache[string(id)]; ok {
		if v.err != nil {
			return nil, v.err
		}
		return v.value, nil
	}

	v, err := f()

	// Only primitives can be cached, allowing us to get out of infinite loop
	if v != nil && v.canCache() {
		cache[string(id)] = spliceValue{err, v}
	}
	return v, err
}
