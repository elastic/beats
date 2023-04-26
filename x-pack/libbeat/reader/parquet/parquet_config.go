// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package parquet

// reader config
type Config struct {
	ProcessParallel bool `config:"process_parallel"`
	BatchSize       int  `config:"batch_size" default:"1"`
}
