// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package parquet

import (
	"os"
	"path"
	"sync"
	"testing"
)

// invocation is a function type that takes a testing.B, a Config, and a file and performs some operation on the file
// this is the common signature for all the benchmark functions in this file
type invocation func(b *testing.B, cfg *Config, file *os.File)

// parquetFile is a struct that contains the name of the parquet
// file to be created and the number of columns and rows in the file
type parquetFile struct {
	name string
	cols int
	rows int
}

func BenchmarkReadParquet(b *testing.B) {
	testCases := []struct {
		desc                       string
		files                      []string
		processParallel            bool
		useGoRoutiunes             bool
		constructAndReadLargeFiles bool
		largeFiles                 []parquetFile
		batchSize                  int
		invokeFn                   invocation
	}{
		{
			desc:      "Process single files serially in batches of 1000",
			files:     []string{"cloudtrail.parquet"},
			batchSize: 1000,
			invokeFn:  readParquetFile,
		},
		{
			desc:      "Process single files serially in batches of 10000",
			files:     []string{"cloudtrail.parquet"},
			batchSize: 10000,
			invokeFn:  readParquetFile,
		},
		{
			desc:            "Process single files parallelly in batches of 1000",
			files:           []string{"cloudtrail.parquet"},
			processParallel: true,
			batchSize:       1000,
			invokeFn:        readParquetFile,
		},
		{
			desc:      "Process single VPC flow file serially in batches of 1000",
			files:     []string{"vpc_flow.gz.parquet"},
			batchSize: 1000,
			invokeFn:  readParquetFile,
		},
		{
			desc:      "Process multiple files serially in batches of 1000",
			files:     []string{"cloudtrail.parquet", "route53.parquet", "vpc_flow.gz.parquet"},
			batchSize: 1000,
			invokeFn:  readParquetFile,
		},
		{
			desc:           "Process multiple files using go routines in batches of 1000",
			files:          []string{"cloudtrail.parquet", "route53.parquet", "vpc_flow.gz.parquet"},
			batchSize:      1000,
			useGoRoutiunes: true,
			invokeFn:       readParquetFile,
		},
		{
			desc:      "Process multiple files serially in batches of 10000",
			files:     []string{"cloudtrail.parquet", "route53.parquet", "vpc_flow.gz.parquet"},
			batchSize: 10000,
			invokeFn:  readParquetFile,
		},
		{
			desc:            "Process multiple files parallelly in batches of 1000",
			files:           []string{"cloudtrail.parquet", "route53.parquet", "vpc_flow.gz.parquet"},
			processParallel: true,
			batchSize:       1000,
			invokeFn:        readParquetFile,
		},
		{
			desc:            "Read a single row from multiple files parallelly",
			files:           []string{"cloudtrail.parquet", "route53.parquet", "vpc_flow.gz.parquet"},
			processParallel: true,
			batchSize:       1,
			invokeFn:        readParquetSingleRow,
		},
		{
			desc:      "Read a single row from multiple files serially",
			files:     []string{"cloudtrail.parquet", "route53.parquet", "vpc_flow.gz.parquet"},
			batchSize: 1,
			invokeFn:  readParquetSingleRow,
		},
		{
			desc:           "Read a single row from multiple files using go routines",
			files:          []string{"cloudtrail.parquet", "route53.parquet", "vpc_flow.gz.parquet"},
			useGoRoutiunes: true,
			batchSize:      1,
			invokeFn:       readParquetSingleRow,
		},
		{
			desc:      "Read a single row from a single file serially",
			files:     []string{"cloudtrail.parquet"},
			batchSize: 1,
			invokeFn:  readParquetSingleRow,
		},
		{
			desc:            "Read a single row from a single file parallelly",
			files:           []string{"cloudtrail.parquet"},
			processParallel: true,
			batchSize:       1,
			invokeFn:        readParquetSingleRow,
		},
		{
			desc:      "Construct a stream reader for a single file serially",
			files:     []string{"cloudtrail.parquet"},
			batchSize: 1,
			invokeFn:  constructBufferedReader,
		},
		{
			desc:            "Construct a stream reader for a single file parallelly",
			files:           []string{"cloudtrail.parquet"},
			processParallel: true,
			batchSize:       1,
			invokeFn:        constructBufferedReader,
		},
		{
			desc:      "Construct a stream reader for multiple files serially",
			files:     []string{"cloudtrail.parquet", "route53.parquet", "vpc_flow.gz.parquet"},
			batchSize: 1,
			invokeFn:  constructBufferedReader,
		},
		{
			desc:            "Construct a stream reader for multiple files parallelly",
			files:           []string{"cloudtrail.parquet", "route53.parquet", "vpc_flow.gz.parquet"},
			processParallel: true,
			batchSize:       1,
			invokeFn:        constructBufferedReader,
		},
		{
			desc:           "Construct a stream reader for multiple files using go routines",
			files:          []string{"cloudtrail.parquet", "route53.parquet", "vpc_flow.gz.parquet"},
			useGoRoutiunes: true,
			batchSize:      1,
			invokeFn:       constructBufferedReader,
		},
		{
			desc:                       "Construct and read a single large parquet file in batches of 1000 serially",
			constructAndReadLargeFiles: true,
			largeFiles: []parquetFile{{
				name: "large_file_1.parquet",
				cols: 4,
				rows: 100000,
			}},
			batchSize: 1000,
			invokeFn:  readParquetFile,
		},
		{
			desc:                       "Construct and read a single large parquet file in batches of 10000 serially",
			constructAndReadLargeFiles: true,
			largeFiles: []parquetFile{{
				name: "large_file_2.parquet",
				cols: 4,
				rows: 100000,
			}},
			batchSize: 10000,
			invokeFn:  readParquetFile,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.desc, func(b *testing.B) {
			cfg := &Config{
				// we set ProcessParallel to true as this always has the best performance
				ProcessParallel: true,
				BatchSize:       tc.batchSize,
			}

			var files []string
			if tc.constructAndReadLargeFiles {
				tempDir := b.TempDir()
				for _, f := range tc.largeFiles {
					fName := tempDir + "/" + f.name
					createRandomParquet(b, fName, f.cols, f.rows)
					files = append(files, fName)
				}
			} else {
				for _, f := range tc.files {
					files = append(files, path.Join(testDataPath, f))

				}
			}

			b.ResetTimer()
			//nolint:errcheck // we do not care about handling errors from file.Seek()
			switch {
			case tc.processParallel:
				b.RunParallel(func(pb *testing.PB) {
					filePtrArr := openFiles(b, files)
					for pb.Next() {
						for _, f := range filePtrArr {
							defer f.Close()
							f.Seek(0, 0)
							tc.invokeFn(b, cfg, f)
						}
					}
				})
			case tc.useGoRoutiunes:
				filePtrArr := openFiles(b, files)
				wg := sync.WaitGroup{}
				for i := 0; i < b.N; i++ {
					for _, f := range filePtrArr {
						defer f.Close()
						f.Seek(0, 0)
						cf := f
						wg.Add(1)
						go func() {
							defer wg.Done()
							tc.invokeFn(b, cfg, cf)
						}()
					}
					wg.Wait()
				}
			// default case is set to serial processing of files
			default:
				filePtrArr := openFiles(b, files)
				for i := 0; i < b.N; i++ {
					for _, f := range filePtrArr {
						defer f.Close()
						f.Seek(0, 0)
						tc.invokeFn(b, cfg, f)
					}
				}
			}
		})
	}
}

// readParquetFile reads entire parquet file
func readParquetFile(b *testing.B, cfg *Config, file *os.File) {
	sReader, err := NewBufferedReader(file, cfg)
	if err != nil {
		b.Fatalf("failed to init stream reader: %v", err)
	}

	for sReader.Next() {
		_, err := sReader.Record()
		if err != nil {
			b.Fatalf("failed to read stream: %v", err)
		}
	}
	err = sReader.Close()
	if err != nil {
		b.Fatalf("failed to close stream reader: %v", err)
	}
}

// readParquetSingleRow reads only the first row of parquet files
func readParquetSingleRow(b *testing.B, cfg *Config, file *os.File) {
	sReader, err := NewBufferedReader(file, cfg)
	if err != nil {
		b.Fatalf("failed to init stream reader: %v", err)
	}

	if sReader.Next() {
		_, err := sReader.Record()
		if err != nil {
			b.Fatalf("failed to read stream: %v", err)
		}
	}
	err = sReader.Close()
	if err != nil {
		b.Fatalf("failed to close stream reader: %v", err)
	}
}

// constructBufferedReader constructs a stream reader for reading parquet files
func constructBufferedReader(b *testing.B, cfg *Config, file *os.File) {
	sReader, err := NewBufferedReader(file, cfg)
	if err != nil {
		b.Fatalf("failed to init stream reader: %v", err)
	}
	err = sReader.Close()
	if err != nil {
		b.Fatalf("failed to close stream reader: %v", err)
	}
}

// openFiles opens parquet files for reading in a slice of file pointers and returns the slice
func openFiles(b *testing.B, files []string) []*os.File {
	filePtrArr := make([]*os.File, len(files))
	for i, f := range files {
		file, err := os.Open(f)
		if err != nil {
			b.Fatalf("failed to open parquet file: %v", err)
		}
		filePtrArr[i] = file
	}
	return filePtrArr
}
