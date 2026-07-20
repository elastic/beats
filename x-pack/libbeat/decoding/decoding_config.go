// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package decoding

// DecoderConfig contains the configuration options for instantiating a decoder.
type DecoderConfig struct {
	Codec *CodecConfig `config:"codec"`
}

// CodecConfig contains the configuration options for different codecs used by a decoder.
type CodecConfig struct {
	Parquet *ParquetCodecConfig `config:"parquet"`
	Auto    *AutoConfig         `config:"auto"`
	JSON    *JSONConfig         `config:"json"`
}

// ParquetCodecConfig contains the configuration options for the parquet codec.
type ParquetCodecConfig struct {
	Enabled         bool `config:"enabled"`
	ProcessParallel bool `config:"process_parallel"`
	BatchSize       int  `config:"batch_size" default:"1"`
}

// AutoConfig contains the configuration options for the auto decoder which uses a set of known codecs to decode the data.
type AutoConfig struct {
	Enabled bool `config:"enabled"`
}

// JSON contains the configuration options for the JSON decoder.
type JSONConfig struct {
	Enabled bool `config:"enabled"`
}
