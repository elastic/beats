// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

// decoderConfig contains the configuration options for instantiating a decoder.
type decoderConfig struct {
	Codec *codecConfig `config:"codec"`
}

// codecConfig contains the configuration options for different codecs used by a decoder.
type codecConfig struct {
	Parquet *parquetCodecConfig `config:"parquet"`
}

// parquetCodecConfig contains the configuration options for the parquet codec.
type parquetCodecConfig struct {
	Enabled         bool `config:"enabled"`
	ProcessParallel bool `config:"process_parallel"`
	BatchSize       int  `config:"batch_size" default:"1"`
}
