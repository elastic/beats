// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/elastic/beats/v7/x-pack/libbeat/reader/ipfix"
)

// decoderConfig contains the configuration options for instantiating a decoder.
type decoderConfig struct {
	Codec *codecConfig `config:"codec"`
}

// codecConfig contains the configuration options for different codecs used by a decoder.
type codecConfig struct {
	Parquet *parquetCodecConfig `config:"parquet"`
	CSV     *csvCodecConfig     `config:"csv"`
	IPFIX   *ipfix.Config       `config:"ipfix"`
}

func (c *codecConfig) Validate() error {
	count := 0
	if c.Parquet != nil {
		count++
	}
	if c.CSV != nil {
		count++
	}
	if c.IPFIX != nil {
		count++
	}

	if count > 1 {
		return errors.New("more than one decoder configured")
	}
	return nil
}

// csvCodecConfig contains the configuration options for the CSV codec.
type csvCodecConfig struct {
	Enabled bool `config:"enabled"`

	// Fields is the set of field names. If it is present
	// it is used to specify the object names of returned
	// values and the FieldsPerRecord field in the csv.Reader.
	// Otherwise, names are obtained from the first
	// line of the CSV data.
	Fields []string `config:"fields_names"`

	// The fields below have the same meaning as the
	// fields of the same name in csv.Reader.
	Comma            *configRune `config:"comma"`
	Comment          configRune  `config:"comment"`
	LazyQuotes       bool        `config:"lazy_quotes"`
	TrimLeadingSpace bool        `config:"trim_leading_space"`
}

type configRune rune

func (r *configRune) Unpack(s string) error {
	if s == "" {
		return nil
	}
	n := utf8.RuneCountInString(s)
	if n != 1 {
		return fmt.Errorf("single character option given more than one character: %q", s)
	}
	_r, _ := utf8.DecodeRuneInString(s)
	*r = configRune(_r)
	return nil
}

// parquetCodecConfig contains the configuration options for the parquet codec.
type parquetCodecConfig struct {
	Enabled         bool `config:"enabled"`
	ProcessParallel bool `config:"process_parallel"`
	BatchSize       int  `config:"batch_size" default:"1"`
}
