// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package input

// Config has options common to all inputs' configurations
type Config struct {
	// KeepNull determines whether published events will keep null values or omit them.
	KeepNull bool `config:"keep_null"`
}
