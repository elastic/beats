// Package ucfg provides a common representation for hierarchical configurations.
//
// The common representation provided by the Config type can be used with different
// configuration file formats like XML, JSON, HSJSON, YAML, or TOML.
//
// Config provides a low level and a high level interface for reading settings
// with additional features like custom unpackers, validation and capturing
// sub-configurations for deferred interpretation, lazy intra-configuration
// variable expansion, and OS environment variable expansion.
package ucfg
