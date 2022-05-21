// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure_blob

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
	"github.com/elastic/beats/v7/libbeat/reader/parser"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	"github.com/pkg/errors"
)

type config struct {
	AccountName      string        `config:"storage_account"`
	AccountKey       string        `config:"storage_account_key"`
	Container        string        `config:"container_name"`
	BlobListPrefix   string        `config:"blob_list_prefix"`
	Endpoint         string        `config:"endpoint"`
	BlobListInterval time.Duration `config:"blob_list_interval"`
	NumberOfWorkers  int           `config:"number_of_workers"`
	ReaderConfig     readerConfig  `config:",inline"` // Reader options to apply when no file_selectors are used.
}

func defaultConfig() config {
	c := config{
		Endpoint:         "blob.core.windows.net",
		BlobListInterval: 120 * time.Second,
		NumberOfWorkers:  1,
		BlobListPrefix:   "",
	}
	c.ReaderConfig.InitDefaults()
	return c
}

func (conf *config) Validate() error {
	if conf.AccountName == "" || conf.AccountKey == "" {
		return errors.New("no storage account or storage account key configured")
	}
	if conf.Container == "" {
		return errors.New("no storage container configured")
	}

	return nil
}

// readerConfig defines the options for reading the content of an S3 object.
type readerConfig struct {
	BufferSize     cfgtype.ByteSize        `config:"buffer_size"`
	ContentType    string                  `config:"content_type"`
	Encoding       string                  `config:"encoding"`
	LineTerminator readfile.LineTerminator `config:"line_terminator"`
	MaxBytes       cfgtype.ByteSize        `config:"max_bytes"`
	Parsers        parser.Config           `config:",inline"`
}

func (rc *readerConfig) Validate() error {
	if rc.BufferSize <= 0 {
		return fmt.Errorf("buffer_size <%v> must be greater than 0", rc.BufferSize)
	}

	if rc.MaxBytes <= 0 {
		return fmt.Errorf("max_bytes <%v> must be greater than 0", rc.MaxBytes)
	}

	_, found := encoding.FindEncoding(rc.Encoding)
	if !found {
		return fmt.Errorf("encoding type <%v> not found", rc.Encoding)
	}

	return nil
}

func (rc *readerConfig) InitDefaults() {
	rc.BufferSize = 16 * humanize.KiByte
	rc.MaxBytes = 10 * humanize.MiByte
	rc.LineTerminator = readfile.AutoLineTerminator
	rc.ContentType = ""
}
