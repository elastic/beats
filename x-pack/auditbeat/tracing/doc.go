// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package tracing provides tools for kprobe-based event tracing. See
// https://www.kernel.org/doc/Documentation/trace/kprobetrace.txt.
// It's built on top of https://github.com/elastic/go-perf which in turn is a fork
// of a candidate package http://golang.org/x/sys/unix/linux/perf currently in review
// at https://go-review.googlesource.com/c/sys/+/168059.
//
// For a higher-level API see the kprobes sub-package at
// http://godoc.org/github.com/elastic/beats/x-pack/auditbeat/tracing/kprobes and
// *kprobes.Engine
package tracing
