// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
