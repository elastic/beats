// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure_blob

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
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

const (
	contentTypeJSON   = "application/json"
	contentTypeNDJSON = "application/x-ndjson"
)

type azureBlobProcessorFactory struct {
	log          *logp.Logger
	metrics      *inputMetrics
	blob         blobGetter
	publisher    beat.Client
	readerConfig readerConfig
}

func newBlobObjectProcessorFactory(log *logp.Logger, metrics *inputMetrics, blob blobGetter, publisher beat.Client, readerConfig readerConfig) *azureBlobProcessorFactory {
	if metrics == nil {
		metrics = newInputMetrics(monitoring.NewRegistry(), "")
	}
	return &azureBlobProcessorFactory{
		log:          log,
		metrics:      metrics,
		blob:         blob,
		publisher:    publisher,
		readerConfig: readerConfig,
	}
}

// Create returns a new azureBlobProcessor. It returns nil when no file selectors
// match the S3 object key.
func (f *azureBlobProcessorFactory) Create(ctx context.Context, log *logp.Logger, ack *EventACKTracker, containerName string, blobName string) *azureBlobProcessor {
	log = log.With(
		"blob_name", blobName)

	return &azureBlobProcessor{
		azureBlobProcessorFactory: f,
		log:                       log,
		ctx:                       ctx,
		acker:                     ack,
		readerConfig:              &f.readerConfig,
		containerName:             containerName,
		blobName:                  blobName,
		blobObjHash:               blobObjectHash(containerName, blobName),
	}
}

type azureBlobProcessor struct {
	*azureBlobProcessorFactory

	log            *logp.Logger
	ctx            context.Context
	acker          *EventACKTracker
	readerConfig   *readerConfig // Config about how to process the object.
	containerName  string
	blobName       string
	blobObjHash    string
	blobRequestURL string
}

func (p *azureBlobProcessor) Wait() {
	p.acker.Wait()
}

func (p *azureBlobProcessor) ProcessBlobObject() error {
	if p == nil {
		return nil
	}

	// Metrics and Logging
	p.log.Debug("Begin Azure blob processing.")
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
	contentType, body, err := p.download()
	if err != nil {
		return fmt.Errorf("failed to get Azure Blob (elapsed_time_ns=%d): %w", time.Since(start).Nanoseconds(), err)
	}
	defer body.Close()
	// p.s3Metadata = meta

	reader, err := p.addGzipDecoderIfNeeded(newMonitoredReader(body, p.metrics.s3BytesProcessedTotal))
	if err != nil {
		return fmt.Errorf("failed checking for gzip content: %w", err)
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
		return fmt.Errorf("failed reading Azure Blob (elapsed_time_ns=%d): %w", time.Since(start).Nanoseconds(), err)
	}

	return nil
}

// download requests the S3 object from AWS and returns the object's
// Content-Type and reader to get the object's contents. The caller must
// close the returned reader.
func (p *azureBlobProcessor) download() (contentType string, body io.ReadCloser, err error) {
	resp, err := p.blob.GetObject(p.ctx, p.blobName)
	if err != nil {
		return "", nil, err
	}

	if resp == nil {
		return "", nil, errors.New("empty response from azure get blob")
	}
	p.blobRequestURL = resp.Response().Request.URL.String()

	if resp.ContentType() == "" {
		return "", resp.Body(azblob.RetryReaderOptions{}), nil
	}
	return resp.ContentType(), resp.Body(azblob.RetryReaderOptions{}), nil
}

func (p *azureBlobProcessor) addGzipDecoderIfNeeded(body io.Reader) (io.Reader, error) {
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

func (p *azureBlobProcessor) readJSON(r io.Reader) error {
	dec := json.NewDecoder(r)
	dec.UseNumber()

	for dec.More() && p.ctx.Err() == nil {
		offset := dec.InputOffset()

		var item json.RawMessage
		if err := dec.Decode(&item); err != nil {
			return fmt.Errorf("failed to decode json: %w", err)
		}

		data, _ := item.MarshalJSON()
		evt := p.createEvent(string(data), offset)
		p.publish(p.acker, &evt)
	}

	return nil
}

func (p *azureBlobProcessor) readFile(r io.Reader) error {
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

		event := p.createEvent(string(message.Content), offset)
		event.Fields.DeepUpdate(message.Fields)
		offset += int64(message.Bytes)
		p.publish(p.acker, &event)
	}

	return nil
}

func (p *azureBlobProcessor) publish(ack *EventACKTracker, event *beat.Event) {
	ack.Add()
	event.Private = ack
	p.metrics.s3EventsCreatedTotal.Inc()
	p.publisher.Publish(*event)
}

func (p *azureBlobProcessor) createEvent(message string, offset int64) beat.Event {
	event := beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: mapstr.M{
			"message": message,
			"log": mapstr.M{
				"offset": offset,
				"file": mapstr.M{
					"path": p.blobRequestURL,
				},
			},
			"azure": mapstr.M{
				"blob": mapstr.M{
					"container": mapstr.M{
						"name": p.containerName,
					},
					"object": mapstr.M{
						"name": p.blobName,
					},
				},
			},
			"cloud": mapstr.M{
				"provider": "azure",
			},
		},
	}
	event.SetID(objectID(p.blobObjHash, offset))

	// if len(p.s3Metadata) > 0 {
	// 	event.Fields.Put("aws.s3.metadata", p.s3Metadata)
	// }

	return event
}

func objectID(objectHash string, offset int64) string {
	return fmt.Sprintf("%s-%012d", objectHash, offset)
}

// blobObjectHash returns a short sha256 hash of the container name + blob name.
func blobObjectHash(containerName string, blobName string) string {
	h := sha256.New()
	h.Write([]byte(containerName))
	h.Write([]byte(blobName))
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
