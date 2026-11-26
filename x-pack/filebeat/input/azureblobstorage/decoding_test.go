// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	azcontainer "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
	"github.com/elastic/beats/v7/libbeat/management/status"
	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/x-pack/libbeat/reader/decoder"
)

// all test files are read from the "testdata" directory
const testDataPath = "testdata"

type noopReporter struct{}

// UpdateStatus is no-op
func (n noopReporter) UpdateStatus(status status.Status, msg string) {}

// statusRecorder records the status updates it receives.
type statusRecorder struct {
	statuses []status.Status
}

func (r *statusRecorder) UpdateStatus(s status.Status, _ string) {
	r.statuses = append(r.statuses, s)
}

func TestDecoding(t *testing.T) {
	log := logptest.NewTestingLogger(t, "")

	testCases := []struct {
		name          string
		file          string
		content       string
		contentType   string
		numEvents     int
		assertAgainst string
		config        decoder.Config
	}{
		{
			name:          "gzip_csv",
			file:          "txn.csv.gz",
			content:       "text/csv",
			numEvents:     4,
			assertAgainst: "txn.json",
			config: decoder.Config{
				Codec: &decoder.CodecConfig{
					CSV: &decoder.CSVCodecConfig{
						Enabled: true,
						Comma:   new(decoder.Rune(' ')),
					},
				},
			},
		},
		{
			name:          "csv",
			file:          "txn.csv",
			content:       "text/csv",
			numEvents:     4,
			assertAgainst: "txn.json",
			config: decoder.Config{
				Codec: &decoder.CodecConfig{
					CSV: &decoder.CSVCodecConfig{
						Enabled: true,
						Comma:   new(decoder.Rune(' ')),
					},
				},
			},
		},
		{
			name:          "csv_quoted_values",
			file:          "txn_quoted.csv",
			content:       "text/csv",
			numEvents:     2,
			assertAgainst: "txn_quoted.json",
			config: decoder.Config{
				Codec: &decoder.CodecConfig{
					CSV: &decoder.CSVCodecConfig{
						Enabled: true,
					},
				},
			},
		},
		{
			name:      "parquet",
			file:      "../../../../libbeat/reader/parquet/testdata/vpc_flow.gz.parquet",
			numEvents: 652,
			config: decoder.Config{
				Codec: &decoder.CodecConfig{
					Parquet: &decoder.ParquetCodecConfig{
						Enabled:   true,
						BatchSize: 1,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(testDataPath, tc.file)
			if tc.contentType == "" {
				tc.contentType = "application/octet-stream"
			}
			f, err := os.Open(file)
			if err != nil {
				t.Fatalf("failed to open test data: %v", err)
			}
			defer f.Close()
			p := &pub{t: t}
			item := &azcontainer.BlobItem{
				Name: new("test_blob"),
				Properties: &azcontainer.BlobProperties{
					ContentType:  new(tc.content),
					LastModified: &time.Time{},
				},
			}
			j := newJob(&blob.Client{}, item, "https://foo.blob.core.windows.net/", newState(), &Source{}, 0, p, noopReporter{}, nil, log)
			j.src.ReaderConfig.Decoding = tc.config
			err = j.decode(context.Background(), f, "test")
			if err != nil {
				t.Errorf("unexpected error calling decode: %v", err)
			}

			events := p.events

			ids := make(map[any]struct{}, len(events))
			for _, event := range events {
				ids[event.Meta["_id"]] = struct{}{}
			}
			assert.Len(t, ids, len(events), "events from the same object must have unique IDs")
			if tc.numEvents != 0 {
				assert.Len(t, events, tc.numEvents, "unexpected number of events")
			}
			if tc.assertAgainst != "" {
				targetData := readJSONFromFile(t, filepath.Join(testDataPath, tc.assertAgainst))
				assert.Equal(t, len(targetData), len(events))

				for i, event := range events {
					msg, err := event.Fields.GetValue("message")
					assert.NoError(t, err)
					assert.JSONEq(t, targetData[i], msg.(string))
				}
			}
		})
	}
}

type pub struct {
	t       *testing.T
	events  []beat.Event
	cursors []any // Non-nil cursors, one for each checkpoint save.
}

func (p *pub) Publish(e beat.Event, cursor any) error {
	p.t.Logf("%v\n", e.Fields)
	p.events = append(p.events, e)
	if cursor != nil {
		p.cursors = append(p.cursors, cursor)
	}
	return nil
}

// readJSONFromFile reads the json file and returns the data as a slice of strings
func readJSONFromFile(t *testing.T, filepath string) []string {
	fileBytes, err := os.ReadFile(filepath)
	assert.NoError(t, err)
	var rawMessages []json.RawMessage
	err = json.Unmarshal(fileBytes, &rawMessages)
	assert.NoError(t, err)
	var data []string

	for _, rawMsg := range rawMessages {
		data = append(data, string(rawMsg))
	}
	return data
}

var codecConfigTests = []struct {
	name    string
	yaml    string
	want    decoder.Config
	wantErr error
}{
	{
		name: "handle_rune",
		yaml: `
codec:
  csv:
    enabled: true
    comma: ' '
    comment: '#'
`,
		want: decoder.Config{
			Codec: &decoder.CodecConfig{
				CSV: &decoder.CSVCodecConfig{
					Enabled: true,
					Comma:   new(decoder.Rune(' ')),
					Comment: '#',
				},
			}},
	},
	{
		name: "no_comma",
		yaml: `
codec:
  csv:
    enabled: true
`,
		want: decoder.Config{
			Codec: &decoder.CodecConfig{
				CSV: &decoder.CSVCodecConfig{
					Enabled: true,
				},
			}},
	},
	{
		name: "null_comma",
		yaml: `
codec:
  csv:
    enabled: true
    comma: "\u0000"
`,
		want: decoder.Config{
			Codec: &decoder.CodecConfig{
				CSV: &decoder.CSVCodecConfig{
					Enabled: true,
					Comma:   new(decoder.Rune('\x00')),
				},
			}},
	},
	{
		name: "bad_rune",
		yaml: `
codec:
  csv:
    enabled: true
    comma: 'this is too long'
`,
		wantErr: errors.New(`single character option given more than one character: "this is too long" accessing 'codec.csv.comma'`),
	},
}

func TestCodecConfig(t *testing.T) {
	for _, test := range codecConfigTests {
		t.Run(test.name, func(t *testing.T) {
			c, err := conf.NewConfigWithYAML([]byte(test.yaml), "")
			if err != nil {
				t.Fatalf("unexpected error unmarshaling config: %v", err)
			}

			var got decoder.Config
			err = c.Unpack(&got)
			if !sameError(err, test.wantErr) {
				t.Errorf("unexpected error unpacking config: got:%v want:%v", err, test.wantErr)
			}

			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("unexpected result\n--- want\n+++ got\n%s", cmp.Diff(test.want, got))
			}
		})
	}
}

func sameError(a, b error) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil, b == nil:
		return false
	default:
		return a.Error() == b.Error()
	}
}

func TestDecodingMaxDecodeBytes(t *testing.T) {
	log := logptest.NewTestingLogger(t, "")

	gz := func(t *testing.T, b []byte) []byte {
		t.Helper()
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, err := zw.Write(b)
		if err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	parquetData, err := os.ReadFile(filepath.Join(testDataPath, "../../../../libbeat/reader/parquet/testdata/vpc_flow.gz.parquet"))
	if err != nil {
		t.Fatalf("failed to read parquet test data: %v", err)
	}
	csvData, err := os.ReadFile(filepath.Join(testDataPath, "txn.csv"))
	if err != nil {
		t.Fatalf("failed to read csv test data: %v", err)
	}

	parquetCodec := decoder.Config{Codec: &decoder.CodecConfig{Parquet: &decoder.ParquetCodecConfig{Enabled: true, BatchSize: 1}}}
	csvCodec := decoder.Config{Codec: &decoder.CodecConfig{CSV: &decoder.CSVCodecConfig{Enabled: true, Comma: new(decoder.Rune(' '))}}}

	tests := []struct {
		name    string
		data    []byte
		codec   decoder.Config
		limit   cfgtype.ByteSize
		wantErr bool
		// noEvents is set for the buffering parquet codec, which must fail
		// before it publishes anything. CSV streams, so may publish some.
		noEvents bool
	}{
		{name: "parquet_over_limit", data: parquetData, codec: parquetCodec, limit: cfgtype.ByteSize(len(parquetData) - 1), wantErr: true, noEvents: true},
		{name: "parquet_at_limit", data: parquetData, codec: parquetCodec, limit: cfgtype.ByteSize(len(parquetData))},
		{name: "parquet_default_limit", data: parquetData, codec: parquetCodec},
		{name: "csv_default_limit", data: csvData, codec: csvCodec},
		// Allow the header and two rows, plus part of a third.
		{name: "csv_over_limit", data: csvData, codec: csvCodec, limit: cfgtype.ByteSize(nthNewline(csvData, 3) + 10), wantErr: true},
		// The limit applies after decompression, so a small compressed blob
		// that inflates beyond the limit is rejected.
		{name: "gzip_csv_inflates_over_limit", data: gz(t, bytes.Repeat(csvData, 100)), codec: csvCodec, limit: cfgtype.ByteSize(len(csvData)), wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := &pub{t: t}
			rec := &statusRecorder{}
			item := &azcontainer.BlobItem{
				Name: new("test_blob"),
				Properties: &azcontainer.BlobProperties{
					ContentType:  new("application/octet-stream"),
					LastModified: &time.Time{},
				},
			}
			j := newJob(&blob.Client{}, item, "https://foo.blob.core.windows.net/", newState(), &Source{}, 0, p, rec, nil, log)
			j.src.ReaderConfig.Decoding = test.codec
			j.src.ReaderConfig.MaxDecodeBytes = test.limit

			err := j.decode(context.Background(), bytes.NewReader(test.data), "test")
			if test.wantErr {
				if !errors.Is(err, errBlobTooLarge) {
					t.Errorf("decode error = %v; want %v", err, errBlobTooLarge)
				}
				if test.noEvents && len(p.events) != 0 {
					t.Errorf("got %d events from oversize blob; want 0", len(p.events))
				}
				if strings.Contains(strings.ToLower(err.Error()), "json") {
					t.Errorf("decode error %q mentions JSON; want an error about the limit", err)
				}
				if n := len(rec.statuses); n == 0 || rec.statuses[n-1] != status.Degraded {
					t.Errorf("status updates = %v; want last to be Degraded", rec.statuses)
				}
				// A blob that was not read completely must be retried, so it
				// must not be checkpointed.
				if len(p.cursors) != 0 {
					t.Errorf("got %d checkpoint saves from oversize blob; want 0", len(p.cursors))
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(p.events) == 0 {
				t.Error("got no events; want some")
			}
			// The parquet decoder's More advances the reader, so it does not
			// report the last row and never saves a checkpoint. That is
			// independent of the limit, so only check the CSV case.
			if test.codec.Codec.CSV != nil && len(p.cursors) != 1 {
				t.Errorf("got %d checkpoint saves; want 1", len(p.cursors))
			}
		})
	}
}

// nthNewline returns the offset of the n'th newline in b.
func nthNewline(b []byte, n int) int {
	off := 0
	for range n {
		i := bytes.IndexByte(b[off:], '\n')
		if i < 0 {
			return len(b)
		}
		off += i + 1
	}
	return off - 1
}

// eofReader returns all of its data and io.EOF in one Read call, as
// compress/gzip does at the end of a stream.
type eofReader struct{ data []byte }

func (r *eofReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	if len(r.data) == 0 {
		return n, io.EOF
	}
	return n, nil
}

func TestLimitedReader(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 100)
	tests := []struct {
		name    string
		src     io.Reader
		max     int64
		wantLen int
		wantErr bool
	}{
		{name: "under", src: bytes.NewReader(data), max: 101, wantLen: 100},
		{name: "exact", src: bytes.NewReader(data), max: 100, wantLen: 100},
		{name: "over_by_one", src: bytes.NewReader(data), max: 99, wantLen: 99, wantErr: true},
		{name: "one_byte_reads", src: iotest.OneByteReader(bytes.NewReader(data)), max: 50, wantLen: 50, wantErr: true},
		{name: "zero_limit", src: bytes.NewReader(data), max: 0, wantLen: 0, wantErr: true},
		{name: "exact_with_eof", src: &eofReader{data: data}, max: 100, wantLen: 100},
		// The overflow must not be hidden by the underlying EOF.
		{name: "over_with_eof", src: &eofReader{data: data}, max: 99, wantLen: 99, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lr := newLimitedReader(test.src, test.max)
			got, err := io.ReadAll(lr)
			if len(got) != test.wantLen {
				t.Errorf("read %d bytes; want %d", len(got), test.wantLen)
			}
			if test.wantErr != errors.Is(err, errBlobTooLarge) {
				t.Errorf("ReadAll error = %v; want errBlobTooLarge: %t", err, test.wantErr)
			}
			if lr.exceeded() != test.wantErr {
				t.Errorf("exceeded() = %t; want %t", lr.exceeded(), test.wantErr)
			}
		})
	}
}

func TestMaxDecodeBytesInheritance(t *testing.T) {
	const globalDecoding = `
decoding.codec.csv.enabled: true
`
	tests := []struct {
		name      string
		global    string
		container string
		wantMax   cfgtype.ByteSize
		wantCSV   bool
	}{
		{name: "none_set", wantCSV: false},
		{name: "global_decoding_only", global: globalDecoding, wantCSV: true},
		{name: "global_limit_only", global: "max_decode_bytes: 5MiB", wantMax: 5 << 20},
		{name: "container_limit_only", container: "max_decode_bytes: 7MiB", wantMax: 7 << 20},
		{
			name:      "container_limit_keeps_global_decoding",
			global:    globalDecoding,
			container: "max_decode_bytes: 7MiB",
			wantMax:   7 << 20, wantCSV: true,
		},
		{
			name:      "container_decoding_keeps_global_limit",
			global:    "max_decode_bytes: 5MiB",
			container: "decoding.codec.csv.enabled: true",
			wantMax:   5 << 20, wantCSV: true,
		},
		{
			name:      "container_limit_overrides_global",
			global:    "max_decode_bytes: 5MiB",
			container: "max_decode_bytes: 7MiB",
			wantMax:   7 << 20,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var y strings.Builder
			y.WriteString("account_name: acct\nauth.shared_credentials.account_key: a2V5\n" + test.global + "\ncontainers:\n  - name: c1\n")
			for l := range strings.SplitSeq(test.container, "\n") {
				if l != "" {
					y.WriteString("    " + l + "\n")
				}
			}
			c, err := conf.NewConfigWithYAML([]byte(y.String()), "")
			if err != nil {
				t.Fatal(err)
			}
			sources, _, err := configure(c, logptest.NewTestingLogger(t, ""))
			if err != nil {
				t.Fatalf("configure error: %v", err)
			}
			if len(sources) != 1 {
				t.Fatalf("got %d sources; want 1", len(sources))
			}
			src, ok := sources[0].(*Source)
			if !ok {
				t.Fatalf("source is %T; want *Source", sources[0])
			}
			rc := src.ReaderConfig
			if rc.MaxDecodeBytes != test.wantMax {
				t.Errorf("MaxDecodeBytes = %d; want %d", rc.MaxDecodeBytes, test.wantMax)
			}
			hasCSV := rc.Decoding.Codec != nil && rc.Decoding.Codec.CSV != nil
			if hasCSV != test.wantCSV {
				t.Errorf("CSV codec configured = %t; want %t", hasCSV, test.wantCSV)
			}
		})
	}
}

// TestMaxDecodeBytesBoundsRead checks the failure mode from the issue: an
// oversize blob must not be buffered into memory. The source is endless, so
// without the limit this would exhaust memory.
func TestMaxDecodeBytesBoundsRead(t *testing.T) {
	log := logptest.NewTestingLogger(t, "")
	const limit = 1 << 20
	cr := &countingReader{}
	item := &azcontainer.BlobItem{
		Name: new("test_blob"),
		Properties: &azcontainer.BlobProperties{
			ContentType:  new("application/octet-stream"),
			LastModified: &time.Time{},
		},
	}
	j := newJob(&blob.Client{}, item, "https://foo.blob.core.windows.net/", newState(), &Source{}, 0, &pub{t: t}, noopReporter{}, nil, log)
	j.src.ReaderConfig.Decoding = decoder.Config{Codec: &decoder.CodecConfig{Parquet: &decoder.ParquetCodecConfig{Enabled: true}}}
	j.src.ReaderConfig.MaxDecodeBytes = limit

	// Bound the source so that a regression fails the test rather than
	// exhausting memory.
	err := j.decode(context.Background(), io.LimitReader(cr, 8*limit), "test")
	if !errors.Is(err, errBlobTooLarge) {
		t.Errorf("decode error = %v; want %v", err, errBlobTooLarge)
	}
	// bufio and the gzip sniffer may read ahead a little, but nowhere near unbounded.
	if cr.n > 2*limit {
		t.Errorf("read %d bytes from source; want at most %d", cr.n, 2*limit)
	}
}

// countingReader is an endless stream of zero bytes that counts what is read.
type countingReader struct{ n int64 }

func (c *countingReader) Read(p []byte) (int, error) {
	clear(p)
	c.n += int64(len(p))
	return len(p), nil
}

func TestMaxDecodeBytesConfig(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    cfgtype.ByteSize
		wantErr bool
	}{
		{name: "unset", yaml: `{}`, want: 0},
		{name: "set", yaml: `max_decode_bytes: 10MiB`, want: 10 << 20},
		{name: "negative", yaml: `max_decode_bytes: -1`, wantErr: true},
		// Overflows int64 to a negative value, which Validate must reject.
		{name: "overflow", yaml: `max_decode_bytes: 9EiB`, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, err := conf.NewConfigWithYAML([]byte(test.yaml), "")
			if err != nil {
				t.Fatal(err)
			}
			var rc readerConfig
			err = c.Unpack(&rc)
			if (err != nil) != test.wantErr {
				t.Fatalf("Unpack error = %v; wantErr %t", err, test.wantErr)
			}
			if err == nil && rc.MaxDecodeBytes != test.want {
				t.Errorf("MaxDecodeBytes = %d; want %d", rc.MaxDecodeBytes, test.want)
			}
			if err == nil && test.want == 0 && decodeLimit(rc) != defaultMaxDecodeBytes {
				t.Errorf("decodeLimit = %d; want default %d", decodeLimit(rc), defaultMaxDecodeBytes)
			}
		})
	}
}
