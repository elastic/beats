package logp

// Option configures the logp package behavior.
type Option func(cfg *Config)

// WithLevel specifies the logging level.
func WithLevel(level Level) Option {
	return func(cfg *Config) {
		cfg.Level = level
	}
}

// WithSelectors specifies what debug selectors are enabled. If no selectors are
// specified then they are all enabled.
func WithSelectors(selectors ...string) Option {
	return func(cfg *Config) {
		cfg.Selectors = append(cfg.Selectors, selectors...)
	}
}

// ToObserverOutput specifies that the output should be collected in memory so
// that they can be read by an observer by calling ObserverLogs().
func ToObserverOutput() Option {
	return func(cfg *Config) {
		cfg.toObserver = true
		cfg.ToStderr = false
	}
}

// ToDiscardOutput configures the logger to write to io.Discard. This is for
// benchmarking purposes only.
func ToDiscardOutput() Option {
	return func(cfg *Config) {
		cfg.toIODiscard = true
		cfg.ToStderr = false
	}
}

// AsJSON specifies to log the output as JSON.
func AsJSON() Option {
	return func(cfg *Config) {
		cfg.JSON = true
	}
}
