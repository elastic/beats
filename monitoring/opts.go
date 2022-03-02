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
