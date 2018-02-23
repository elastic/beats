// Package prospector allows to define new way of reading data in Filebeat
// Deprecated: See the input package
package prospector

import "github.com/elastic/beats/filebeat/input"

// Prospectorer defines how to read new data
// Deprecated: See input.input
type Prospectorer = input.Input

// Runner encapsulate the lifecycle of a prospectorer
// Deprecated: See input.Runner
type Runner = input.Runner

// Context wrapper for backward compatibility
// Deprecated: See input.Context
type Context = input.Context

// Factory wrapper for backward compatibility
// Deprecated: See input.Factory
type Factory = input.Factory

// Register wrapper for backward compatibility
// Deprecated: See input.Register
var Register = input.Register

// GetFactory wrapper for backward compatibility
// Deprecated: See input.GetFactory
var GetFactory = input.GetFactory

// New wrapper for backward compatibility
// Deprecated: see input.New
var New = input.New

// NewRunnerFactory wrapper for backward compatibility
// Deprecated: see input.NewRunnerFactory
var NewRunnerFactory = input.NewRunnerFactory
