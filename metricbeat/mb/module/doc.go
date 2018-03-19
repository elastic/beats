// Package module contains the low-level utilities for running Metricbeat
// modules and metricsets. This is useful for building your own tool that
// has a module and sub-module concept. If you want to reuse the whole
// Metricbeat framework see the github.com/elastic/beats/metricbeat/beater
// package that provides a higher level interface.
//
// This contains the tools for instantiating modules, running them, and
// connecting their outputs to the Beat's output pipeline.
package module
