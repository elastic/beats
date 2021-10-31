// Copyright 2016 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bigquery

import (
	"context"

	"cloud.google.com/go/internal/trace"
	bq "google.golang.org/api/bigquery/v2"
)

// ExtractConfig holds the configuration for an extract job.
type ExtractConfig struct {
	// Src is the table from which data will be extracted.
	// Only one of Src or SrcModel should be specified.
	Src *Table

	// SrcModel is the ML model from which the data will be extracted.
	// Only one of Src or SrcModel should be specified.
	SrcModel *Model

	// Dst is the destination into which the data will be extracted.
	Dst *GCSReference

	// DisableHeader disables the printing of a header row in exported data.
	DisableHeader bool

	// The labels associated with this job.
	Labels map[string]string

	// For Avro-based extracts, controls whether logical type annotations are generated.
	//
	// Example:  With this enabled, writing a BigQuery TIMESTAMP column will result in
	// an integer column annotated with the appropriate timestamp-micros/millis annotation
	// in the resulting Avro files.
	UseAvroLogicalTypes bool
}

func (e *ExtractConfig) toBQ() *bq.JobConfiguration {
	var printHeader *bool
	if e.DisableHeader {
		f := false
		printHeader = &f
	}
	cfg := &bq.JobConfiguration{
		Labels: e.Labels,
		Extract: &bq.JobConfigurationExtract{
			DestinationUris:   append([]string{}, e.Dst.URIs...),
			Compression:       string(e.Dst.Compression),
			DestinationFormat: string(e.Dst.DestinationFormat),
			FieldDelimiter:    e.Dst.FieldDelimiter,

			PrintHeader:         printHeader,
			UseAvroLogicalTypes: e.UseAvroLogicalTypes,
		},
	}
	if e.Src != nil {
		cfg.Extract.SourceTable = e.Src.toBQ()
	}
	if e.SrcModel != nil {
		cfg.Extract.SourceModel = e.SrcModel.toBQ()
	}
	return cfg
}

func bqToExtractConfig(q *bq.JobConfiguration, c *Client) *ExtractConfig {
	qe := q.Extract
	return &ExtractConfig{
		Labels: q.Labels,
		Dst: &GCSReference{
			URIs:              qe.DestinationUris,
			Compression:       Compression(qe.Compression),
			DestinationFormat: DataFormat(qe.DestinationFormat),
			FileConfig: FileConfig{
				CSVOptions: CSVOptions{
					FieldDelimiter: qe.FieldDelimiter,
				},
			},
		},
		DisableHeader:       qe.PrintHeader != nil && !*qe.PrintHeader,
		Src:                 bqToTable(qe.SourceTable, c),
		SrcModel:            bqToModel(qe.SourceModel, c),
		UseAvroLogicalTypes: qe.UseAvroLogicalTypes,
	}
}

// An Extractor extracts data from a BigQuery table into Google Cloud Storage.
type Extractor struct {
	JobIDConfig
	ExtractConfig
	c *Client
}

// ExtractorTo returns an Extractor which can be used to extract data from a
// BigQuery table into Google Cloud Storage.
// The returned Extractor may optionally be further configured before its Run method is called.
func (t *Table) ExtractorTo(dst *GCSReference) *Extractor {
	return &Extractor{
		c: t.c,
		ExtractConfig: ExtractConfig{
			Src: t,
			Dst: dst,
		},
	}
}

// ExtractorTo returns an Extractor which can be persist a BigQuery Model into
// Google Cloud Storage.
// The returned Extractor may be further configured before its Run method is called.
func (m *Model) ExtractorTo(dst *GCSReference) *Extractor {
	return &Extractor{
		c: m.c,
		ExtractConfig: ExtractConfig{
			SrcModel: m,
			Dst:      dst,
		},
	}
}

// Run initiates an extract job.
func (e *Extractor) Run(ctx context.Context) (j *Job, err error) {
	ctx = trace.StartSpan(ctx, "cloud.google.com/go/bigquery.Extractor.Run")
	defer func() { trace.EndSpan(ctx, err) }()

	return e.c.insertJob(ctx, e.newJob(), nil)
}

func (e *Extractor) newJob() *bq.Job {
	return &bq.Job{
		JobReference:  e.JobIDConfig.createJobRef(e.c),
		Configuration: e.ExtractConfig.toBQ(),
	}
}
