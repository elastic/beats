package monitoring

// Option type for passing additional options to NewRegistry.
type Option func(options) options

type options struct {
	publishExpvar bool
	mode          Mode
}

var defaultOptions = options{
	publishExpvar: false,
	mode:          Full,
}

// PublishExpvar enables publishing all registered variables via expvar interface.
// Note: expvar does not allow removal of any stats.
func PublishExpvar(o options) options {
	o.publishExpvar = true
	return o
}

// IgnorePublishExpvar disables publishing expvar variables in a sub-registry.
func IgnorePublishExpvar(o options) options {
	o.publishExpvar = false
	return o
}

func Report(o options) options {
	o.mode = Reported
	return o
}

func DoNotReport(o options) options {
	o.mode = Full
	return o
}

func varOpts(regOpts *options, opts []Option) *options {
	if regOpts != nil && len(opts) == 0 {
		return regOpts
	}

	O := defaultOptions
	if regOpts != nil {
		O = *regOpts
	}

	for _, opt := range opts {
		O = opt(O)
	}
	return &O
}

func applyOpts(in *options, opts []Option) *options {
	if len(opts) == 0 {
		return ensureOptions(in)
	}

	tmp := *ensureOptions(in)
	for _, opt := range opts {
		tmp = opt(tmp)
	}
	return &tmp
}

func ensureOptions(in *options) *options {
	if in != nil {
		return in
	}

	tmp := defaultOptions
	return &tmp
}
