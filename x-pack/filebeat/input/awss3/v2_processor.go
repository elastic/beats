// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	x_reader "github.com/elastic/beats/v7/x-pack/libbeat/reader"
	"github.com/elastic/beats/v7/x-pack/libbeat/reader/decoder"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// objectProcessorV2 downloads and decodes an S3 object into beat.Events.
// Unlike the legacy processor, finalization (backup/delete) is decoupled
// and triggered externally after all events have been ACKed.
type objectProcessorV2 struct {
	s3      s3Getter
	metrics *inputMetrics

	fileSelectors []fileSelectorConfig
	backupConfig  backupConfig
}

func newObjectProcessorV2(s3 s3Getter, metrics *inputMetrics, selectors []fileSelectorConfig, backup backupConfig) *objectProcessorV2 {
	if len(selectors) == 0 {
		selectors = []fileSelectorConfig{
			{ReaderConfig: defaultConfig().ReaderConfig},
		}
	}
	return &objectProcessorV2{
		s3:            s3,
		metrics:       metrics,
		fileSelectors: selectors,
		backupConfig:  backup,
	}
}

// ProcessObject downloads and decodes the S3 object into events. It calls
// publish for each event produced. It returns the number of events emitted
// and any processing error.
//
// A returned error wrapping errS3DownloadFailed indicates a transient
// download problem (the caller should retry). Other errors are permanent.
func (op *objectProcessorV2) ProcessObject(ctx context.Context, log *logp.Logger, obj s3EventV2, pub func(beat.Event)) (int, error) {
	rc := op.findReaderConfig(obj.S3.Object.Key)
	if rc == nil {
		return 0, nil
	}

	log = log.With(
		"bucket_arn", obj.S3.Bucket.ARN,
		"object_key", obj.S3.Object.Key)

	op.metrics.s3ObjectsRequestedTotal.Inc()
	op.metrics.s3ObjectsInflight.Inc()
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		op.metrics.s3ObjectsInflight.Dec()
		op.metrics.s3ObjectProcessingTime.Update(elapsed.Nanoseconds())
		log.Debugw("Object processing complete.", "elapsed_time_ns", elapsed)
	}()

	downloaded, err := op.download(ctx, obj, rc)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errS3DownloadFailed, err)
	}
	defer downloaded.body.Close()

	mReader := newMonitoredReader(downloaded.body, op.metrics.s3BytesProcessedTotal)
	wrappedReader := s3DownloadFailedWrappedReader{r: mReader}

	streamReader, err := x_reader.AddGzipDecoderIfNeeded(wrappedReader)
	if err != nil {
		return 0, fmt.Errorf("failed checking for gzip content: %w", err)
	}

	if rc.ContentType != "" {
		downloaded.contentType = rc.ContentType
	}

	objHash := objectHashV2(obj)
	builder := &eventBuilderV2{
		obj:        obj,
		hash:       objHash,
		requestURL: downloaded.requestURL,
		metadata:   downloaded.metadata,
	}

	var eventCount int
	emit := func(e beat.Event) {
		eventCount++
		pub(e)
	}

	dec, err := decoder.NewDecoder(rc.Decoding, streamReader, log)
	if err != nil {
		return 0, err
	}

	switch dec := dec.(type) {
	case decoder.ValueDecoder:
		defer dec.Close()
		for dec.Next() {
			evtOffset, msg, _, err := dec.DecodeValue()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return eventCount, fmt.Errorf("failed reading s3 object: %w", err)
			}
			emit(builder.newEvent(string(msg), evtOffset))
		}

	case decoder.Decoder:
		defer dec.Close()
		var evtOffset int64
		for dec.Next() {
			data, err := dec.Decode()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return eventCount, fmt.Errorf("failed reading s3 object: %w", err)
			}
			evtOffset, err = op.readJSONSlice(builder, emit, rc, bytes.NewReader(data), evtOffset)
			if err != nil {
				return eventCount, err
			}
		}

	default:
		switch {
		case strings.HasPrefix(downloaded.contentType, contentTypeJSON) || strings.HasPrefix(downloaded.contentType, contentTypeNDJSON):
			err = op.readJSON(ctx, builder, emit, rc, streamReader)
		default:
			err = op.readFile(builder, emit, rc, streamReader, log)
		}
		if err != nil {
			return eventCount, fmt.Errorf("failed reading s3 object (elapsed_time_ns=%d): %w",
				time.Since(start).Nanoseconds(), err)
		}
	}

	op.metrics.s3ObjectSizeInBytes.Update(mReader.totalBytesReadCurrent)
	return eventCount, nil
}

func (op *objectProcessorV2) findReaderConfig(key string) *readerConfig {
	for _, sel := range op.fileSelectors {
		if sel.Regex == nil || sel.Regex.MatchString(key) {
			return &sel.ReaderConfig
		}
	}
	return nil
}

type downloadedObject struct {
	body        io.ReadCloser
	contentType string
	requestURL  string
	metadata    map[string]interface{}
}

func (op *objectProcessorV2) download(ctx context.Context, obj s3EventV2, rc *readerConfig) (*downloadedObject, error) {
	out, err := op.s3.GetObject(ctx, obj.AWSRegion, obj.S3.Bucket.Name, obj.S3.Object.Key)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, fmt.Errorf("empty response from S3 GetObject")
	}

	reqURL := ""
	if v, ok := out.ResultMetadata.Get(s3RequestURLMetadataKey).(string); ok {
		reqURL = v
	}

	ctType := ""
	if out.ContentType != nil {
		ctType = *out.ContentType
	}

	return &downloadedObject{
		body:        out.Body,
		contentType: ctType,
		requestURL:  reqURL,
		metadata:    s3Metadata(out, rc.IncludeS3Metadata...),
	}, nil
}

// readJSON decodes a JSON stream into events, handling expand_event_list_from_field.
func (op *objectProcessorV2) readJSON(ctx context.Context, bld *eventBuilderV2, pub func(beat.Event), rc *readerConfig, r io.Reader) error {
	dec := json.NewDecoder(r)
	dec.UseNumber()

	for dec.More() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		offset := dec.InputOffset()

		var item json.RawMessage
		if err := dec.Decode(&item); err != nil {
			return fmt.Errorf("failed to decode json: %w", err)
		}

		if rc.ExpandEventListFromField != "" {
			if err := op.splitEventList(bld, pub, rc.ExpandEventListFromField, item, offset); err != nil {
				return err
			}
			continue
		}

		data, _ := item.MarshalJSON()
		pub(bld.newEvent(string(data), offset))
	}
	return nil
}

// readJSONSlice processes a JSON array where each element is a separate event.
func (op *objectProcessorV2) readJSONSlice(bld *eventBuilderV2, pub func(beat.Event), rc *readerConfig, r io.Reader, offset int64) (int64, error) {
	dec := json.NewDecoder(r)
	dec.UseNumber()

	_, err := dec.Token()
	if err != nil {
		return -1, fmt.Errorf("failed to read JSON slice opening token: %w", err)
	}

	for dec.More() {
		var item json.RawMessage
		if err := dec.Decode(&item); err != nil {
			return -1, fmt.Errorf("failed to decode json: %w", err)
		}

		if rc.ExpandEventListFromField != "" {
			if err := op.splitEventList(bld, pub, rc.ExpandEventListFromField, item, offset); err != nil {
				return -1, err
			}
			continue
		}

		data, _ := item.MarshalJSON()
		pub(bld.newEvent(string(data), offset))
		offset++
	}
	return offset, nil
}

func (op *objectProcessorV2) splitEventList(bld *eventBuilderV2, pub func(beat.Event), key string, raw json.RawMessage, offset int64) error {
	if key != ".[]" {
		var jsonObject map[string]json.RawMessage
		if err := json.Unmarshal(raw, &jsonObject); err != nil {
			return err
		}
		var found bool
		raw, found = jsonObject[key]
		if !found {
			return fmt.Errorf("expand_event_list_from_field key <%v> is not in event", key)
		}
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '[' {
		return fmt.Errorf("expand_event_list_from_field <%v> is not an array", key)
	}

	for dec.More() {
		arrayOffset := dec.InputOffset()

		var item json.RawMessage
		if err := dec.Decode(&item); err != nil {
			return fmt.Errorf("failed to decode array item at offset %d: %w", offset+arrayOffset, err)
		}

		data, _ := item.MarshalJSON()
		pub(bld.newEvent(string(data), offset+arrayOffset))
	}
	return nil
}

// readFile processes the object as line-oriented text using the encoding/parser chain.
func (op *objectProcessorV2) readFile(bld *eventBuilderV2, pub func(beat.Event), rc *readerConfig, r io.Reader, log *logp.Logger) error {
	encoderFrom, ok := encoding.FindEncoding(rc.Encoding)
	if !ok || encoderFrom == nil {
		return fmt.Errorf("failed to find '%v' encoding", rc.Encoding)
	}

	enc, err := encoderFrom(r)
	if err != nil {
		return fmt.Errorf("failed to initialize encoding: %w", err)
	}

	var rd reader.Reader
	rd, err = readfile.NewEncodeReader(io.NopCloser(r), readfile.Config{
		Codec:        enc,
		BufferSize:   int(rc.BufferSize),
		Terminator:   rc.LineTerminator,
		CollectOnEOF: true,
		MaxBytes:     int(rc.MaxBytes) * 4,
	}, log)
	if err != nil {
		return fmt.Errorf("failed to create encode reader: %w", err)
	}

	rd = readfile.NewStripNewline(rd, rc.LineTerminator)
	rd = rc.Parsers.Create(rd, log)
	rd = readfile.NewLimitReader(rd, int(rc.MaxBytes))

	var offset int64
	for {
		message, err := rd.Next()
		if len(message.Content) > 0 {
			event := bld.newEvent(string(message.Content), offset)
			event.Fields.DeepUpdate(message.Fields)
			offset += int64(message.Bytes)
			pub(event)
		}

		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading message: %w", err)
		}
	}
	return nil
}

// Finalize copies the object to backup and optionally deletes it.
// This is called after all events from the object have been ACKed.
func (op *objectProcessorV2) Finalize(ctx context.Context, s3Move s3Mover, obj s3EventV2) error {
	bucketName := op.backupConfig.GetBucketName()
	if bucketName == "" {
		return nil
	}
	backupKey := op.backupConfig.BackupToBucketPrefix + obj.S3.Object.Key
	_, err := s3Move.CopyObject(ctx, obj.AWSRegion, obj.S3.Bucket.Name, bucketName, obj.S3.Object.Key, backupKey)
	if err != nil {
		return fmt.Errorf("failed to copy object to backup bucket: %w", err)
	}
	if !op.backupConfig.Delete {
		return nil
	}
	_, err = s3Move.DeleteObject(ctx, obj.AWSRegion, obj.S3.Bucket.Name, obj.S3.Object.Key)
	if err != nil {
		return fmt.Errorf("failed to delete object from bucket: %w", err)
	}
	return nil
}

// eventBuilderV2 constructs beat.Events from object metadata and message content.
type eventBuilderV2 struct {
	obj        s3EventV2
	hash       string
	requestURL string
	metadata   map[string]interface{}
}

func (b *eventBuilderV2) newEvent(message string, offset int64) beat.Event {
	event := beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: mapstr.M{
			"message": message,
			"log": mapstr.M{
				"file": mapstr.M{
					"path": b.requestURL,
				},
			},
			"aws": mapstr.M{
				"s3": mapstr.M{
					"bucket": mapstr.M{
						"name": b.obj.S3.Bucket.Name,
						"arn":  b.obj.S3.Bucket.ARN,
					},
					"object": mapstr.M{
						"key": b.obj.S3.Object.Key,
					},
				},
			},
			"cloud": mapstr.M{
				"provider": b.obj.Provider,
				"region":   b.obj.AWSRegion,
			},
		},
	}
	if offset >= 0 {
		event.Fields.Put("log.offset", offset)
		event.SetID(fmt.Sprintf("%d-%s-%012d", b.obj.S3.Object.LastModified.UnixNano(), b.hash, offset))
	}
	if len(b.metadata) > 0 {
		event.Fields.Put("aws.s3.metadata", b.metadata)
	}
	return event
}

func objectHashV2(obj s3EventV2) string {
	h := sha256.New()
	h.Write([]byte(obj.S3.Bucket.ARN))
	h.Write([]byte(obj.S3.Object.Key))
	prefix := hex.EncodeToString(h.Sum(nil))
	return prefix[:10]
}
