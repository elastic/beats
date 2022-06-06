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
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
)

const (
	contentTypeJSON   = "application/json"
	contentTypeNDJSON = "application/x-ndjson"
)

type s3ObjectProcessorFactory struct {
	log           *logp.Logger
	metrics       *inputMetrics
	s3            s3Getter
	publisher     beat.Client
	fileSelectors []fileSelectorConfig
}

func newS3ObjectProcessorFactory(log *logp.Logger, metrics *inputMetrics, s3 s3Getter, publisher beat.Client, sel []fileSelectorConfig) *s3ObjectProcessorFactory {
	if metrics == nil {
		metrics = newInputMetrics(monitoring.NewRegistry(), "")
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
		publisher:     publisher,
		fileSelectors: sel,
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
func (f *s3ObjectProcessorFactory) Create(ctx context.Context, log *logp.Logger, ack *eventACKTracker, obj s3EventV2) s3ObjectHandler {
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
		acker:                    ack,
		readerConfig:             readerConfig,
		s3Obj:                    obj,
		s3ObjHash:                s3ObjectHash(obj),
	}
}

type s3ObjectProcessor struct {
	*s3ObjectProcessorFactory

	log          *logp.Logger
	ctx          context.Context
	acker        *eventACKTracker // ACKer tied to the SQS message (multiple S3 readers share an ACKer when the S3 notification event contains more than one S3 object).
	readerConfig *readerConfig    // Config about how to process the object.
	s3Obj        s3EventV2        // S3 object information.
	s3ObjHash    string

	s3Metadata map[string]interface{} // S3 object metadata.
}

func (p *s3ObjectProcessor) Wait() {
	p.acker.Wait()
}

func (p *s3ObjectProcessor) ProcessS3Object() error {
	if p == nil {
		return nil
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
		return errors.Wrapf(err, "failed to get s3 object (elasped_time_ns=%d)",
			time.Since(start).Nanoseconds())
	}
	defer body.Close()
	p.s3Metadata = meta

	reader, err := p.addGzipDecoderIfNeeded(newMonitoredReader(body, p.metrics.s3BytesProcessedTotal))
	if err != nil {
		return errors.Wrap(err, "failed checking for gzip content")
	}

	// Overwrite with user configured Content-Type.
	if p.readerConfig.ContentType != "" {
		contentType = p.readerConfig.ContentType
	}

	// Process object content stream.
	switch {
	case contentType == contentTypeJSON || contentType == contentTypeNDJSON:
		err = p.readJSON(reader)
	default:
		err = p.readFile(reader)
	}
	if err != nil {
		return errors.Wrapf(err, "failed reading s3 object (elasped_time_ns=%d)",
			time.Since(start).Nanoseconds())
	}

	return nil
}

// download requests the S3 object from AWS and returns the object's
// Content-Type and reader to get the object's contents. The caller must
// close the returned reader.
func (p *s3ObjectProcessor) download() (contentType string, metadata map[string]interface{}, body io.ReadCloser, err error) {
	resp, err := p.s3.GetObject(p.ctx, p.s3Obj.S3.Bucket.Name, p.s3Obj.S3.Object.Key)
	if err != nil {
		return "", nil, nil, err
	}

	if resp == nil {
		return "", nil, nil, errors.New("empty response from s3 get object")
	}

	meta := s3Metadata(resp, p.readerConfig.IncludeS3Metadata...)
	if resp.ContentType == nil {
		return "", meta, resp.Body, nil
	}
	return *resp.ContentType, meta, resp.Body, nil
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
		evt := createEvent(string(data), offset, p.s3Obj, p.s3ObjHash, p.s3Metadata)
		p.publish(p.acker, &evt)
	}

	return nil
}

func (p *s3ObjectProcessor) splitEventList(key string, raw json.RawMessage, offset int64, objHash string) error {
	var jsonObject map[string]json.RawMessage
	if err := json.Unmarshal(raw, &jsonObject); err != nil {
		return err
	}

	raw, found := jsonObject[key]
	if !found {
		return fmt.Errorf("expand_event_list_from_field key <%v> is not in event", key)
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
		evt := createEvent(string(data), offset+arrayOffset, p.s3Obj, objHash, p.s3Metadata)
		p.publish(p.acker, &evt)
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
	reader, err = readfile.NewEncodeReader(ioutil.NopCloser(r), readfile.Config{
		Codec:      enc,
		BufferSize: int(p.readerConfig.BufferSize),
		Terminator: p.readerConfig.LineTerminator,
		MaxBytes:   int(p.readerConfig.MaxBytes) * 4,
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
		if err == io.EOF {
			// No more lines
			break
		}
		if err != nil {
			return fmt.Errorf("error reading message: %w", err)
		}

		event := createEvent(string(message.Content), offset, p.s3Obj, p.s3ObjHash, p.s3Metadata)
		event.Fields.DeepUpdate(message.Fields)
		offset += int64(message.Bytes)
		p.publish(p.acker, &event)
	}

	return nil
}

func (p *s3ObjectProcessor) publish(ack *eventACKTracker, event *beat.Event) {
	ack.Add()
	event.Private = ack
	p.metrics.s3EventsCreatedTotal.Inc()
	p.publisher.Publish(*event)
}

func createEvent(message string, offset int64, obj s3EventV2, objectHash string, meta map[string]interface{}) beat.Event {
	event := beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: common.MapStr{
			"message": message,
			"log": common.MapStr{
				"offset": offset,
				"file": common.MapStr{
					"path": constructObjectURL(obj),
				},
			},
			"aws": common.MapStr{
				"s3": common.MapStr{
					"bucket": common.MapStr{
						"name": obj.S3.Bucket.Name,
						"arn":  obj.S3.Bucket.ARN},
					"object": common.MapStr{
						"key": obj.S3.Object.Key,
					},
				},
			},
			"cloud": common.MapStr{
				"provider": "aws",
				"region":   obj.AWSRegion,
			},
		},
	}
	event.SetID(objectID(objectHash, offset))

	if len(meta) > 0 {
		event.Fields.Put("aws.s3.metadata", meta)
	}

	return event
}

func objectID(objectHash string, offset int64) string {
	return fmt.Sprintf("%s-%012d", objectHash, offset)
}

func constructObjectURL(obj s3EventV2) string {
	return "https://" + obj.S3.Bucket.Name + ".s3." + obj.AWSRegion + ".amazonaws.com/" + obj.S3.Object.Key
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
func s3Metadata(resp *s3.GetObjectResponse, keys ...string) common.MapStr {
	if len(keys) == 0 {
		return nil
	}

	// When you upload objects using the REST API, the optional user-defined
	// metadata names must begin with "x-amz-meta-" to distinguish them from
	// other HTTP headers.
	const userMetaPrefix = "x-amz-meta-"

	allMeta := map[string]interface{}{}

	// Get headers using AWS SDK struct tags.
	fields := reflect.TypeOf(resp.GetObjectOutput).Elem()
	values := reflect.ValueOf(resp.GetObjectOutput).Elem()
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
	metadata := common.MapStr{}
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
