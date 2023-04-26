// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//nolint:errcheck // It's a test file, we don't care about errors here.
package parquet

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func BenchmarkReadParquetSingleSerialBatch_1000(b *testing.B) {
	f := "testdata/taxi_2023_1.parquet"

	cfg := &Config{
		ProcessParallel: true,
		BatchSize:       1000,
	}

	path, _ := filepath.Abs(f)
	file, err := os.Open(path)
	if err != nil {
		b.Fatalf("failed to open parquet file: %v\n", err)
	}
	defer file.Close()

	fn, size := getFileData(b, file)
	var batches int
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		file.Seek(0, 0)
		batches = readParquet(b, cfg, file)
	}

	b.Logf("file: %s, size: %dB,  approx_records: %d, batches: %d \n", fn, size, batches*cfg.BatchSize, batches)
}

func BenchmarkReadParquetSingleSerialBatch_10000(b *testing.B) {
	f := "testdata/taxi_2023_1.parquet"

	cfg := &Config{
		ProcessParallel: true,
		BatchSize:       10000,
	}

	path, _ := filepath.Abs(f)
	file, err := os.Open(path)
	if err != nil {
		b.Fatalf("failed to open parquet file: %v\n", err)
	}
	defer file.Close()

	fn, size := getFileData(b, file)
	var batches int
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		file.Seek(0, 0)
		batches = readParquet(b, cfg, file)
	}

	b.Logf("file: %s, size: %dB,  approx_records: %d, batches: %d \n", fn, size, batches*cfg.BatchSize, batches)
}

func BenchmarkReadParquetSingleVPCSerialBatch_1000(b *testing.B) {
	f := "testdata/vpc_flow.gz.parquet"

	cfg := &Config{
		ProcessParallel: true,
		BatchSize:       1000,
	}

	path, _ := filepath.Abs(f)
	file, err := os.Open(path)
	if err != nil {
		b.Fatalf("failed to open parquet file: %v\n", err)
	}
	defer file.Close()

	fn, size := getFileData(b, file)
	var batches int
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		file.Seek(0, 0)
		batches = readParquet(b, cfg, file)
	}

	b.Logf("file: %s, size: %dB,  approx_records: %d, batches: %d \n", fn, size, batches*cfg.BatchSize, batches)
}

func BenchmarkReadParquetMultiSerialBatch_1000(b *testing.B) {
	farr := []string{
		"testdata/taxi_2023_1.parquet",
		"testdata/taxi_2023_2.parquet",
		"testdata/vpc_flow.gz.parquet",
	}

	cfg := &Config{
		ProcessParallel: true,
		BatchSize:       1000,
	}

	filePtrArr := make([]*os.File, len(farr))
	for i, f := range farr {
		path, _ := filepath.Abs(f)
		file, err := os.Open(path)
		if err != nil {
			b.Fatalf("failed to open parquet file: %v\n", err)
		}
		defer file.Close()
		filePtrArr[i] = file
	}
	batchMap := make(map[string]int)
	for i := 0; i < b.N; i++ {
		for _, f := range filePtrArr {
			f.Seek(0, 0)
			batches := readParquet(b, cfg, f)
			batchMap[f.Name()] = batches
		}
	}

	for _, f := range filePtrArr {
		fn, size := getFileData(b, f)
		batches := batchMap[f.Name()]
		b.Logf("file: %s, size: %dB,  approx_records: %d, batches: %d \n", fn, size, batches*cfg.BatchSize, batches)
	}
}

func BenchmarkReadParquetMultiGoRoutineBatch_1000(b *testing.B) {
	farr := []string{
		"testdata/taxi_2023_1.parquet",
		"testdata/taxi_2023_2.parquet",
		"testdata/vpc_flow.gz.parquet",
	}

	cfg := &Config{
		ProcessParallel: true,
		BatchSize:       1000,
	}

	wg := sync.WaitGroup{}
	filePtrArr := make([]*os.File, len(farr))
	for i, f := range farr {
		path, _ := filepath.Abs(f)
		file, err := os.Open(path)
		if err != nil {
			b.Fatalf("failed to open parquet file: %v\n", err)
		}
		defer file.Close()
		filePtrArr[i] = file
	}

	for i := 0; i < b.N; i++ {
		for _, f := range filePtrArr {
			f.Seek(0, 0)
			wg.Add(1)
			go readParquetPrallel(b, cfg, f, &wg)
		}
		wg.Wait()
	}

	for _, f := range filePtrArr {
		fn, size := getFileData(b, f)
		b.Logf("file: %s, size: %dB\n", fn, size)
	}
}

func BenchmarkReadParquetMultiSerialBatch_10000(b *testing.B) {
	farr := []string{
		"testdata/taxi_2023_1.parquet",
		"testdata/taxi_2023_2.parquet",
		"testdata/vpc_flow.gz.parquet",
	}

	cfg := &Config{
		ProcessParallel: true,
		BatchSize:       10000,
	}

	filePtrArr := make([]*os.File, len(farr))
	for i, f := range farr {
		path, _ := filepath.Abs(f)
		file, err := os.Open(path)
		if err != nil {
			b.Fatalf("failed to open parquet file: %v\n", err)
		}
		defer file.Close()
		filePtrArr[i] = file
	}

	batchMap := make(map[string]int)
	for i := 0; i < b.N; i++ {
		for _, f := range filePtrArr {
			f.Seek(0, 0)
			batches := readParquet(b, cfg, f)
			batchMap[f.Name()] = batches
		}
	}

	for _, f := range filePtrArr {
		fn, size := getFileData(b, f)
		batches := batchMap[f.Name()]
		b.Logf("file: %s, size: %dB,  approx_records: %d, batches: %d \n", fn, size, batches*cfg.BatchSize, batches)
	}
}

func BenchmarkReadParquetSingleParallelBatch_1000(b *testing.B) {
	f := "testdata/taxi_2023_1.parquet"
	cfg := &Config{
		ProcessParallel: true,
		BatchSize:       1000,
	}

	path, _ := filepath.Abs(f)
	file, err := os.Open(path)
	if err != nil {
		b.Fatalf("failed to open parquet file: %v\n", err)
	}
	defer file.Close()

	fn, size := getFileData(b, file)
	var batches int
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			file.Seek(0, 0)
			batches = readParquet(b, cfg, file)
		}
	})

	b.Logf("file: %s, size: %dB,  approx_records: %d, batches: %d \n", fn, size, batches*cfg.BatchSize, batches)
}

func BenchmarkReadParquetSingleParallelBatch_10000(b *testing.B) {
	f := "testdata/taxi_2023_1.parquet"
	cfg := &Config{
		ProcessParallel: true,
		BatchSize:       10000,
	}

	path, _ := filepath.Abs(f)
	file, err := os.Open(path)
	if err != nil {
		b.Fatalf("failed to open parquet file: %v\n", err)
	}
	defer file.Close()

	fn, size := getFileData(b, file)
	var batches int
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			file.Seek(0, 0)
			batches = readParquet(b, cfg, file)
		}
	})

	b.Logf("file: %s, size: %dB,  approx_records: %d, batches: %d \n", fn, size, batches*cfg.BatchSize, batches)
}

func readParquet(t testing.TB, cfg *Config, file *os.File) int {
	count := 0
	sReader, err := NewStreamReader(file, cfg)
	require.NoError(t, err)

	for sReader.Next() {
		_, err := sReader.Read()
		require.NoError(t, err)
		count++
	}

	return count
}

func readParquetPrallel(t testing.TB, cfg *Config, file *os.File, wg *sync.WaitGroup) {
	sReader, err := NewStreamReader(file, cfg)
	require.NoError(t, err)

	for sReader.Next() {
		_, err := sReader.Read()
		require.NoError(t, err)
	}

	wg.Done()
}

func getFileData(t testing.TB, file *os.File) (string, int) {
	fi, err := file.Stat()
	size := 0
	if err != nil {
		t.Logf("could not obtain file stat: %v\n", err)
		return "", size
	} else {
		size = int(fi.Size())
	}
	fn := strings.Split(file.Name(), "/")
	return fn[len(fn)-1], size
}
