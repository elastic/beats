// Package v2 proviades common lumberjack protocol version 2 definitions.
package v2

// Version declares the protocol revision supported by this package.
const Version = 2

// Lumberjack protocol version 2 message types.
const (
	CodeVersion byte = '2'

	CodeWindowSize    byte = 'W'
	CodeJSONDataFrame byte = 'J'
	CodeCompressed    byte = 'C'
	CodeACK           byte = 'A'
)
