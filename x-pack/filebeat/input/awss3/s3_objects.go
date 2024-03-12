// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	contentTypeJSON   = "application/json"
	contentTypeNDJSON = "application/x-ndjson"
)

type s3ObjectProcessorFactory struct {
	log           *logp.Logger
	metrics       *inputMetrics
	s3            s3API
	fileSelectors []fileSelectorConfig
	backupConfig  backupConfig
}

func newS3ObjectProcessorFactory(log *logp.Logger, metrics *inputMetrics, s3 s3API, sel []fileSelectorConfig, backupConfig backupConfig, maxWorkers int) *s3ObjectProcessorFactory {
	if metrics == nil {
		// Metrics are optional. Initialize a stub.
		metrics = newInputMetrics("", nil, 0)
	}
	if len(sel) == 0 {
		sel = []fileSelectorConfig{
			{ReaderConfig: defaultConfig().ReaderConfig},
		}
	}
	return &s3ObjectProcessorFactory{
		log:           log,
		metrics:       metrics,
		s3:            s3,
		fileSelectors: sel,
		backupConfig:  backupConfig,
	}
}

func (f *s3ObjectProcessorFactory) findReaderConfig(key string) *readerConfig {
	for _, sel := range f.fileSelectors {
		if sel.Regex == nil || sel.Regex.MatchString(key) {
			return &sel.ReaderConfig
		}
	}
	return nil
}

// Create returns a new s3ObjectProcessor. It returns nil when no file selectors
// match the S3 object key.
func (f *s3ObjectProcessorFactory) Create(ctx context.Context, log *logp.Logger, client beat.Client, ack *EventACKTracker, obj s3EventV2) s3ObjectHandler {
	log = log.With(
		"bucket_arn", obj.S3.Bucket.Name,
		"object_key", obj.S3.Object.Key)

	readerConfig := f.findReaderConfig(obj.S3.Object.Key)
	if readerConfig == nil {
		log.Debug("Skipping S3 object processing. No file_selectors are a match.")
		return nil
	}

	return &s3ObjectProcessor{
		s3ObjectProcessorFactory: f,
		log:                      log,
		ctx:                      ctx,
		publisher:                client,
		acker:                    ack,
		readerConfig:             readerConfig,
		s3Obj:                    obj,
		s3ObjHash:                s3ObjectHash(obj),
	}
}

// CreateForS3Polling returns a new s3ObjectProcessor. It returns nil when no file selectors
// match the S3 object key.
func (f *s3ObjectProcessorFactory) CreateForS3Polling(ctx context.Context, log *logp.Logger, client beat.Client, ack *awscommon.EventACKTracker, obj s3EventV2) s3ObjectHandler {
	log = log.With(
		"bucket_arn", obj.S3.Bucket.Name,
		"object_key", obj.S3.Object.Key)

	readerConfig := f.findReaderConfig(obj.S3.Object.Key)
	if readerConfig == nil {
		log.Debug("Skipping S3 object processing. No file_selectors are a match.")
		return nil
	}

	return &s3ObjectProcessor{
		s3ObjectProcessorFactory: f,
		log:                      log,
		ctx:                      ctx,
		publisher:                client,
		ackerForPolling:          ack,
		readerConfig:             readerConfig,
		s3Obj:                    obj,
		s3ObjHash:                s3ObjectHash(obj),
	}
}

type s3ObjectProcessor struct {
	*s3ObjectProcessorFactory

	log             *logp.Logger
	ctx             context.Context
	publisher       beat.Client
	acker           *EventACKTracker           // ACKer tied to the SQS message (multiple S3 readers share an ACKer when the S3 notification event contains more than one S3 object).
	ackerForPolling *awscommon.EventACKTracker // ACKer tied to the S3 object (multiple S3 readers share an ACKer when the S3 notification event contains more than one S3 object).
	readerConfig    *readerConfig              // Config about how to process the object.
	s3Obj           s3EventV2                  // S3 object information.
	s3ObjHash       string
	s3RequestURL    string

	s3Metadata map[string]interface{} // S3 object metadata.

	eventsPublishedTotal uint64
}

func (p *s3ObjectProcessor) Wait() {
	p.ackerForPolling.Wait()
}

func (p *s3ObjectProcessor) ProcessS3Object() (uint64, error) {
	if p == nil {
		return 0, nil
	}

	// Metrics and Logging
	p.log.Debug("Begin S3 object processing.")
	p.metrics.s3ObjectsRequestedTotal.Inc()
	p.metrics.s3ObjectsInflight.Inc()
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		p.metrics.s3ObjectsInflight.Dec()
		p.metrics.s3ObjectProcessingTime.Update(elapsed.Nanoseconds())
		p.log.Debugw("End S3 object processing.", "elapsed_time_ns", elapsed)
	}()

	// Request object (download).
	contentType, meta, body, err := p.download()
	if err != nil {
		return 0, fmt.Errorf("failed to get s3 object (elapsed_time_ns=%d): %w",
			time.Since(start).Nanoseconds(), err)
	}
	defer body.Close()
	p.s3Metadata = meta

	reader, err := p.addGzipDecoderIfNeeded(newMonitoredReader(body, p.metrics.s3BytesProcessedTotal))
	if err != nil {
		return 0, fmt.Errorf("failed checking for gzip content: %w", err)
	}

	// Overwrite with user configured Content-Type.
	if p.readerConfig.ContentType != "" {
		contentType = p.readerConfig.ContentType
	}

	// try to create a decoder from the using the codec config
	decoder, decoderErr := newDecoder(p.readerConfig.Decoding, reader)
	if decoderErr != nil {
		return 0, err
	}
	if decoder != nil {
		defer decoder.close()

		var evtOffset int64
		for decoder.next() {
			data, decoderErr := decoder.decode()
			if decoderErr != nil {
				if errors.Is(err, io.EOF) {
					return p.eventsPublishedTotal, nil
				}
				break
			}
			evtOffset, err = p.readJSONSlice(bytes.NewReader(data), evtOffset)
			if err != nil {
				break
			}
		}
	} else {
		// This is the legacy path. It will be removed in future and clubbed together with the decoder.
		// Process object content stream.
		switch {
		case strings.HasPrefix(contentType, contentTypeJSON) || strings.HasPrefix(contentType, contentTypeNDJSON):
			err = p.readJSON(reader)
		default:
			err = p.readFile(reader)
		}
	}
	if err != nil {
		return 0, fmt.Errorf("failed reading s3 object (elapsed_time_ns=%d): %w",
			time.Since(start).Nanoseconds(), err)
	}

	return p.eventsPublishedTotal, nil
}

// download requests the S3 object from AWS and returns the object's
// Content-Type and reader to get the object's contents. The caller must
// close the returned reader.
func (p *s3ObjectProcessor) download() (contentType string, metadata map[string]interface{}, body io.ReadCloser, err error) {
	getObjectOutput, err := p.s3.GetObject(p.ctx, p.s3Obj.S3.Bucket.Name, p.s3Obj.S3.Object.Key)
	if err != nil {
		return "", nil, nil, err
	}

	if getObjectOutput == nil {
		return "", nil, nil, fmt.Errorf("empty response from s3 get object: %w", err)
	}
	s3RequestURL := getObjectOutput.ResultMetadata.Get(s3RequestURLMetadataKey)
	if s3RequestURLAsString, ok := s3RequestURL.(string); ok {
		p.s3RequestURL = s3RequestURLAsString
	}

	meta := s3Metadata(getObjectOutput, p.readerConfig.IncludeS3Metadata...)
	if getObjectOutput.ContentType == nil {
		return "", meta, getObjectOutput.Body, nil
	}
	return *getObjectOutput.ContentType, meta, getObjectOutput.Body, nil
}

func (p *s3ObjectProcessor) addGzipDecoderIfNeeded(body io.Reader) (io.Reader, error) {
	bufReader := bufio.NewReader(body)

	gzipped, err := isStreamGzipped(bufReader)
	if err != nil {
		return nil, err
	}
	if !gzipped {
		return bufReader, nil
	}

	return gzip.NewReader(bufReader)
}

func (p *s3ObjectProcessor) readJSON(r io.Reader) error {
	dec := json.NewDecoder(r)
	dec.UseNumber()

	for dec.More() && p.ctx.Err() == nil {
		offset := dec.InputOffset()

		var item json.RawMessage
		if err := dec.Decode(&item); err != nil {
			return fmt.Errorf("failed to decode json: %w", err)
		}

		if p.readerConfig.ExpandEventListFromField != "" {
			if err := p.splitEventList(p.readerConfig.ExpandEventListFromField, item, offset, p.s3ObjHash); err != nil {
				return err
			}
			continue
		}

		data, _ := item.MarshalJSON()
		evt := p.createEvent(string(data), offset)
		p.publish(&evt)
	}

	return nil
}

// readJSONSlice uses a json.RawMessage to process JSON slice data as individual JSON objects.
// It accepts a reader and a starting offset, it returns an updated offset and an error if any.
// It reads the opening token separately and then iterates over the slice, decoding each object and publishing it.
func (p *s3ObjectProcessor) readJSONSlice(r io.Reader, evtOffset int64) (int64, error) {
	dec := json.NewDecoder(r)
	dec.UseNumber()

	// reads starting token separately since this is always a slice.
	_, err := dec.Token()
	if err != nil {
		return -1, fmt.Errorf("failed to read JSON slice token for object key: %s, with error: %w", p.s3Obj.S3.Object.Key, err)
	}

	// we track each event offset separately since we are reading a slice.
	for dec.More() && p.ctx.Err() == nil {
		var item json.RawMessage
		if err := dec.Decode(&item); err != nil {
			return -1, fmt.Errorf("failed to decode json: %w", err)
		}

		if p.readerConfig.ExpandEventListFromField != "" {
			if err := p.splitEventList(p.readerConfig.ExpandEventListFromField, item, evtOffset, p.s3ObjHash); err != nil {
				return -1, err
			}
			continue
		}

		data, _ := item.MarshalJSON()
		evt := p.createEvent(string(data), evtOffset)
		p.publish(&evt)
		evtOffset++
	}

	return evtOffset, p.ctx.Err()
}

func (p *s3ObjectProcessor) splitEventList(key string, raw json.RawMessage, offset int64, objHash string) error {
	// .[] signifies the root object is an array, and it should be split.
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
		p.s3ObjHash = objHash
		evt := p.createEvent(string(data), offset+arrayOffset)
		p.publish(&evt)
	}

	return nil
}

func (p *s3ObjectProcessor) readFile(r io.Reader) error {
	encodingFactory, ok := encoding.FindEncoding(p.readerConfig.Encoding)
	if !ok || encodingFactory == nil {
		return fmt.Errorf("failed to find '%v' encoding", p.readerConfig.Encoding)
	}

	enc, err := encodingFactory(r)
	if err != nil {
		return fmt.Errorf("failed to initialize encoding: %w", err)
	}

	var reader reader.Reader
	reader, err = readfile.NewEncodeReader(io.NopCloser(r), readfile.Config{
		Codec:        enc,
		BufferSize:   int(p.readerConfig.BufferSize),
		Terminator:   p.readerConfig.LineTerminator,
		CollectOnEOF: true,
		MaxBytes:     int(p.readerConfig.MaxBytes) * 4,
	})
	if err != nil {
		return fmt.Errorf("failed to create encode reader: %w", err)
	}

	reader = readfile.NewStripNewline(reader, p.readerConfig.LineTerminator)
	reader = p.readerConfig.Parsers.Create(reader)
	reader = readfile.NewLimitReader(reader, int(p.readerConfig.MaxBytes))

	var offset int64
	for {
		message, err := reader.Next()
		if len(message.Content) > 0 {
			event := p.createEvent(string(message.Content), offset)
			event.Fields.DeepUpdate(message.Fields)
			offset += int64(message.Bytes)
			p.publish(&event)
		}

		if errors.Is(err, io.EOF) {
			// No more lines
			break
		}
		if err != nil {
			return fmt.Errorf("error reading message: %w", err)
		}
	}

	return nil
}

func (p *s3ObjectProcessor) publish(event *beat.Event) {
	if p.acker != nil {
		event.Private = p.acker
	} else if p.ackerForPolling != nil {
		p.ackerForPolling.Add()
		event.Private = p.ackerForPolling
	}

	p.eventsPublishedTotal++
	p.metrics.s3EventsCreatedTotal.Inc()
	p.publisher.Publish(*event)
}

func (p *s3ObjectProcessor) createEvent(message string, offset int64) beat.Event {
	event := beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: mapstr.M{
			"message": message,
			"log": mapstr.M{
				"offset": offset,
				"file": mapstr.M{
					"path": p.s3RequestURL,
				},
			},
			"aws": mapstr.M{
				"s3": mapstr.M{
					"bucket": mapstr.M{
						"name": p.s3Obj.S3.Bucket.Name,
						"arn":  p.s3Obj.S3.Bucket.ARN,
					},
					"object": mapstr.M{
						"key": p.s3Obj.S3.Object.Key,
					},
				},
			},
			"cloud": mapstr.M{
				"provider": p.s3Obj.Provider,
				"region":   p.s3Obj.AWSRegion,
			},
		},
	}
	event.SetID(objectID(p.s3ObjHash, offset))

	if len(p.s3Metadata) > 0 {
		_, _ = event.Fields.Put("aws.s3.metadata", p.s3Metadata)
	}

	return event
}

func (p *s3ObjectProcessor) FinalizeS3Object() error {
	bucketName := p.backupConfig.GetBucketName()
	if bucketName == "" {
		return nil
	}
	backupKey := p.s3Obj.S3.Object.Key
	if p.backupConfig.BackupToBucketPrefix != "" {
		backupKey = fmt.Sprintf("%s%s", p.backupConfig.BackupToBucketPrefix, backupKey)
	}
	_, err := p.s3.CopyObject(p.ctx, p.s3Obj.S3.Bucket.Name, bucketName, p.s3Obj.S3.Object.Key, backupKey)
	if err != nil {
		return fmt.Errorf("failed to copy object to backup bucket: %w", err)
	}
	if !p.backupConfig.Delete {
		return nil
	}
	_, err = p.s3.DeleteObject(p.ctx, p.s3Obj.S3.Bucket.Name, p.s3Obj.S3.Object.Key)
	if err != nil {
		return fmt.Errorf("failed to delete object from bucket: %w", err)
	}
	return nil
}

func objectID(objectHash string, offset int64) string {
	return fmt.Sprintf("%s-%012d", objectHash, offset)
}

// s3ObjectHash returns a short sha256 hash of the bucket arn + object key name.
func s3ObjectHash(obj s3EventV2) string {
	h := sha256.New()
	h.Write([]byte(obj.S3.Bucket.ARN))
	h.Write([]byte(obj.S3.Object.Key))
	prefix := hex.EncodeToString(h.Sum(nil))
	return prefix[:10]
}

// isStreamGzipped determines whether the given stream of bytes (encapsulated in a buffered reader)
// represents gzipped content or not. A buffered reader is used so the function can peek into the byte
// stream without consuming it. This makes it convenient for code executed after this function call
// to consume the stream if it wants.
func isStreamGzipped(r *bufio.Reader) (bool, error) {
	buf, err := r.Peek(3)
	if err != nil && err != io.EOF {
		return false, err
	}

	// gzip magic number (1f 8b) and the compression method (08 for DEFLATE).
	return bytes.HasPrefix(buf, []byte{0x1F, 0x8B, 0x08}), nil
}

// s3Metadata returns a map containing the selected S3 object metadata keys.
func s3Metadata(resp *s3.GetObjectOutput, keys ...string) mapstr.M {
	if len(keys) == 0 {
		return nil
	}

	// When you upload objects using the REST API, the optional user-defined
	// metadata names must begin with "x-amz-meta-" to distinguish them from
	// other HTTP headers.
	const userMetaPrefix = "x-amz-meta-"

	allMeta := map[string]interface{}{}

	// Get headers using AWS SDK struct tags.
	fields := reflect.TypeOf(resp).Elem()
	values := reflect.ValueOf(resp).Elem()
	for i := 0; i < fields.NumField(); i++ {
		f := fields.Field(i)

		if loc, _ := f.Tag.Lookup("location"); loc != "header" {
			continue
		}

		name, found := f.Tag.Lookup("locationName")
		if !found {
			continue
		}
		name = strings.ToLower(name)

		if name == userMetaPrefix {
			continue
		}

		v := values.Field(i)
		switch v.Kind() {
		case reflect.Ptr:
			if v.IsNil() {
				continue
			}
			v = v.Elem()
		default:
			if v.IsZero() {
				continue
			}
		}

		allMeta[name] = v.Interface()
	}

	// Add in the user defined headers.
	for k, v := range resp.Metadata {
		k = strings.ToLower(k)
		allMeta[userMetaPrefix+k] = v
	}

	// Select the matching headers from the config.
	metadata := mapstr.M{}
	for _, key := range keys {
		key = strings.ToLower(key)

		v, found := allMeta[key]
		if !found {
			continue
		}

		metadata[key] = v
	}

	return metadata
}
