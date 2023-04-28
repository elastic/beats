// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package parquet

// Config contains the parquet reader config options.
type Config struct {
	// If ProcessParallel is true, then functions which read multiple columns will read those columns in parallel
	// from the file with a number of readers equal to the number of columns. Otherwise columns are read serially.
	ProcessParallel bool `config:"process_parallel"`
	// BatchSize is the number of rows to read at a time from the file.
	BatchSize int `config:"batch_size" default:"1"`
}
