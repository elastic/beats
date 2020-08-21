// Package registries provides utility functions for wrapping	beats
// functionality into v2.Registries in order to discovery and create inputs
// based on legacy code.
package registries

//go:generate godocdown -plain=false -output Readme.md

// XXX: if possible we should strive to get rid of these helpers.
// XXX: The interface filebeat/input/v2.Registry was introduced to ease the creation of the wrappers.
