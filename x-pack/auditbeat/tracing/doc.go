// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package tracing provides a set of tools built on top of
// golang.org/x/sys/unix/linux/perf that simplify working with KProbes and
// UProbes, using tracing perf channels to receive events from the kernel and
// decoding of this raw events into more useful types.
package tracing
