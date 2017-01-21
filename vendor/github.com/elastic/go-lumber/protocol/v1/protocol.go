// Package v1 proviades common lumberjack protocol version 1 definitions.
package v1

// Version declares the protocol revision supported by this package.
const Version = 1

// Lumberjack protocol version 1 message types.
const (
	CodeVersion byte = '1'

	CodeWindowSize byte = 'W'
	CodeDataFrame  byte = 'D'
	CodeCompressed byte = 'C'
	CodeACK        byte = 'A'
)
