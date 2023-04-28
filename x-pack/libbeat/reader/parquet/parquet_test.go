// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package parquet

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/apache/arrow/go/arrow/memory"
	"github.com/apache/arrow/go/v11/arrow"
	"github.com/apache/arrow/go/v11/arrow/array"
	"github.com/apache/arrow/go/v11/parquet/pqarrow"
	"gotest.tools/assert"
)

// all test files are read from/stored within the "testdata" directory
const testDataPath = "testdata/"

// test file used for reading/writing temporary parquet data
const testFile = "test.parquet"

func TestParquet(t *testing.T) {
	testCases := []struct {
		columns int
		rows    int
	}{
		{
			columns: 10,
			rows:    20,
		},
		{
			columns: 15,
			rows:    30,
		},
		{
			columns: 5,
			rows:    50,
		},
		{
			columns: 10,
			rows:    1000,
		},
		{
			columns: 19,
			rows:    10000,
		},
		{
			columns: 25,
			rows:    10000,
		},
	}

	// cleanup process in case of abrupt exit
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		<-sigc
		os.Remove(testDataPath + testFile)
		os.Exit(1)
	}()

	for _, tc := range testCases {
		name := fmt.Sprintf("Test parquet files with rows=%d, and columns=%d", tc.rows, tc.columns)
		t.Run(name, func(t *testing.T) {
			fName := testDataPath + testFile
			data := createRandomParquet(t, fName, tc.columns, tc.rows)
			file, err := os.Open(fName)
			if err != nil {
				t.Fatalf("Failed to open parquet test file: %v", err)
			}
			defer file.Close()
			defer os.Remove(fName)

			// we set a timeout to prevent the test from running forever
			// 10 minutes should be more than enough for any test case with rows * cols < 1000000
			timeout := time.NewTimer(10 * time.Minute)
			t.Cleanup(func() { timeout.Stop() })

			cfg := &Config{
				// we set ProcessParallel to true as this always has the best performance
				ProcessParallel: true,
				// batch size is set to 1 because we need to compare individual records one by one
				BatchSize: 1,
			}
			rows := readAndValidateParquetFile(t, cfg, file, data)
			// asserts of number of rows read is the same as the number of rows written
			assert.Equal(t, rows, tc.rows)
		})
	}
}

// readAndValidateParquetFile reads the parquet file and validates the data
func readAndValidateParquetFile(t *testing.T, cfg *Config, file *os.File, data map[string]bool) int {
	sReader, err := NewStreamReader(file, cfg)
	if err != nil {
		t.Fatalf("failed to init stream reader: %v\n", err)
	}

	rowCount := 0
	for sReader.Next() {
		val, err := sReader.Record()
		if err != nil {
			t.Fatalf("failed to read stream: %v\n", err)
		}
		if val != nil {
			rowCount++
			// this is where we check if the column values are the same as the ones we wrote
			if _, ok := data[string(val)]; !ok {
				t.Fatalf("failed to find record in parquet file: %v\n", err)
			}
		}
	}
	return rowCount
}

// createRandomParquet creates a parquet file with random data
func createRandomParquet(t testing.TB, fname string, numCols int, numRows int) map[string]bool {
	// defines a map to store the parquet data for validation
	data := make(map[string]bool)
	// creates a new Arrow schema
	var fields []arrow.Field
	for i := 0; i < numCols; i++ {
		fieldType := arrow.PrimitiveTypes.Int32
		field := arrow.Field{Name: fmt.Sprintf("col%d", i), Type: fieldType, Nullable: true}
		fields = append(fields, field)
	}
	schema := arrow.NewSchema(fields, nil)
	file, err := os.Create(fname)
	if err != nil {
		t.Fatalf("Failed to create parquet test file: %v", err)
	}
	defer file.Close()

	// creates a new file writer
	fileWriter, err := pqarrow.NewFileWriter(schema, file, nil, pqarrow.ArrowWriterProperties{})
	if err != nil {
		t.Fatalf("Failed to create parquet file writer: %v", err)
	}
	defer fileWriter.Close()

	// creates an Arrow memory pool for managing memory allocations
	memoryPool := memory.NewGoAllocator()

	// generates random data for writing to the parquet file
	for rowIdx := int64(0); rowIdx < int64(numRows); rowIdx++ {
		// creates an Arrow record with random data
		var recordColumns []arrow.Array
		for colIdx := 0; colIdx < numCols; colIdx++ {
			randData := make([]int32, 1)
			randData[0] = rand.Int31()
			builder := array.NewInt32Builder(memoryPool)
			builder.AppendValues(randData, nil)
			defer builder.Release()
			columnArray := array.NewInt32Data(builder.NewArray().Data())
			recordColumns = append(recordColumns, columnArray)
		}
		record := array.NewRecord(schema, recordColumns, 1)
		defer record.Release()
		val, err := record.MarshalJSON()
		if err != nil {
			t.Fatalf("Failed to marshal record to JSON: %v", err)
		}
		data[string(val)] = true

		// writes the record batch to the Parquet file
		err = fileWriter.Write(record)
		if err != nil {
			t.Fatalf("Failed to write record to parquet file: %v", err)
		}
	}

	return data
}
