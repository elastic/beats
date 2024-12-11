// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

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
	"io"
	"time"
	"unicode"

	"cloud.google.com/go/storage"

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type job struct {
	// gcs bucket handle
	bucket *storage.BucketHandle
	// gcs object attribute struct
	object *storage.ObjectAttrs
	// gcs uri for the resource
	objectURI string
	// object hash, used in setting event id
	hash string
	// flag to denote if object is gZipped compressed or not.
	isCompressed bool
	// flag to denote if object's root element is of an array type
	isRootArray bool
	// object state
	state *state
	// bucket source struct used for storing bucket related data
	src *Source
	// publisher is used to publish a beat event to the output stream
	publisher cursor.Publisher
	// metrics used to track the errors and success of jobs
	metrics *inputMetrics
	// custom logger
	log *logp.Logger
	// flag used to denote if this object has previously failed without being processed at all.
	isFailed bool
}

// newJob, returns an instance of a job, which is a unit of work that can be assigned to a go routine
func newJob(bucket *storage.BucketHandle, object *storage.ObjectAttrs, objectURI string,
	state *state, src *Source, publisher cursor.Publisher, metrics *inputMetrics, log *logp.Logger, isFailed bool,
) *job {
	if metrics == nil {
		// metrics are optional, initialize a stub if not provided
		metrics = newInputMetrics("", nil)
	}
	return &job{
		bucket:    bucket,
		object:    object,
		objectURI: objectURI,
		hash:      gcsObjectHash(src, object),
		state:     state,
		src:       src,
		publisher: publisher,
		metrics:   metrics,
		log:       log,
		isFailed:  isFailed,
	}
}

// gcsObjectHash returns a short sha256 hash of the bucket name + object name.
func gcsObjectHash(src *Source, object *storage.ObjectAttrs) string {
	h := sha256.New()
	h.Write([]byte(src.BucketName))
	h.Write([]byte((object.Name)))
	return hex.EncodeToString(h.Sum(nil)[:5])
}

func (j *job) do(ctx context.Context, id string) {
	var fields mapstr.M
	// metrics & logging
	j.log.Debug("begin gcs object processing.")
	j.metrics.gcsObjectsRequestedTotal.Inc()
	j.metrics.gcsObjectsInflight.Inc()
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		j.metrics.gcsObjectsInflight.Dec()
		j.metrics.gcsObjectProcessingTime.Update(elapsed.Nanoseconds())
		j.log.Debugw("end gcs object processing.", "elapsed_time_ns", elapsed)
	}()

	if allowedContentTypes[j.object.ContentType] {
		if j.object.ContentType == gzType || j.object.ContentEncoding == encodingGzip {
			j.isCompressed = true
		}
		err := j.processAndPublishData(ctx, id)
		if err != nil {
			j.state.updateFailedJobs(j.object.Name, j.metrics)
			j.log.Errorw("job encountered an error while publishing data and has been added to a failed jobs list", "gcs.jobId", id, "error", err)
			j.metrics.gcsFailedJobsTotal.Inc()
			j.metrics.errorsTotal.Inc()
			return
		}
		j.metrics.gcsObjectsPublishedTotal.Inc()
		//nolint:gosec // object size cannot be negative hence this conversion is safe
		j.metrics.gcsBytesProcessedTotal.Add(uint64(j.object.Size))

	} else {
		err := fmt.Errorf("job with jobId %s encountered an error: content-type %s not supported", id, j.object.ContentType)
		fields = mapstr.M{
			"message": err.Error(),
		}
		event := beat.Event{
			Timestamp: time.Now(),
			Fields:    fields,
		}
		event.SetID(objectID(j.hash, 0))
		// locks while data is being saved and published to avoid concurrent map read/writes
		cp, done := j.state.saveForTx(j.object.Name, j.object.Updated, j.metrics)
		if err := j.publisher.Publish(event, cp); err != nil {
			j.log.Errorw("job encountered an error while publishing event", "gcs.jobId", id, "error", err)
			j.metrics.errorsTotal.Inc()
		}
		// unlocks after data is saved and published
		done()
	}
}

func (j *job) Name() string {
	return j.object.Name
}

func (j *job) Source() *Source {
	return j.src
}

func (j *job) Timestamp() time.Time {
	return j.object.Updated
}

func (j *job) processAndPublishData(ctx context.Context, id string) error {
	obj := j.bucket.Object(j.object.Name)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to open reader for object: %s, with error: %w", j.object.Name, err)
	}
	defer func() {
		err = reader.Close()
		if err != nil {
			j.metrics.errorsTotal.Inc()
			j.log.Errorw("failed to close reader for object", "objectName", j.object.Name, "error", err)
		}
	}()

	// update the source lag time metric
	j.metrics.sourceLagTime.Update(time.Since(j.object.Updated).Nanoseconds())

	// calculate number of decode errors
	if err := j.decode(ctx, reader, id); err != nil {
		j.metrics.decodeErrorsTotal.Inc()
		return fmt.Errorf("failed to decode object: %s, with error: %w", j.object.Name, err)
	}

	return nil
}

func (j *job) decode(ctx context.Context, r io.Reader, id string) error {
	r, err := j.addGzipDecoderIfNeeded(bufio.NewReader(r))
	if err != nil {
		return fmt.Errorf("failed to add gzip decoder to object: %s, with error: %w", j.object.Name, err)
	}
	dec, err := newDecoder(j.src.ReaderConfig.Decoding, r)
	if err != nil {
		return err
	}
	var evtOffset int64
	switch dec := dec.(type) {
	case valueDecoder:
		defer dec.close()

		for dec.next() {
			var (
				msg []byte
				val []mapstr.M
			)
			if j.src.ParseJSON {
				var v mapstr.M
				msg, v, err = dec.decodeValue()
				if err != nil {
					if err == io.EOF {
						return nil
					}
					break
				}
				val = []mapstr.M{v}
			} else {
				msg, err = dec.decode()
				if err != nil {
					if err == io.EOF {
						return nil
					}
					break
				}
			}
			evt := j.createEvent(msg, val, evtOffset)
			j.publish(evt, !dec.more(), id)
		}

	case decoder:
		defer dec.close()

		for dec.next() {
			msg, err := dec.decode()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				break
			}
			var val []mapstr.M
			if j.src.ParseJSON {
				val, err = decodeJSON(bytes.NewReader(msg))
				if err != nil {
					j.log.Errorw("job encountered an error", "gcs.jobId", id, "error", err)
				}
			}
			evt := j.createEvent(msg, val, evtOffset)
			j.publish(evt, !dec.more(), id)
		}

	default:
		err = j.readJsonAndPublish(ctx, r, id)
		if err != nil {
			return fmt.Errorf("failed to read data from object: %s, with error: %w", j.object.Name, err)
		}
	}

	return err
}

func (j *job) readJsonAndPublish(ctx context.Context, r io.Reader, id string) error {
	var err error
	r, j.isRootArray, err = evaluateJSON(bufio.NewReader(r))
	if err != nil {
		return fmt.Errorf("failed to evaluate json for object: %s, with error: %w", j.object.Name, err)
	}

	dec := json.NewDecoder(r)
	// UseNumber causes the Decoder to unmarshal a number into an interface{} as a Number instead of as a float64.
	dec.UseNumber()
	// If array is present at root then read json token and advance decoder
	if j.isRootArray {
		_, err := dec.Token()
		if err != nil {
			return fmt.Errorf("failed to read JSON token for object: %s, with error: %w", j.object.Name, err)
		}
	}

	for dec.More() && ctx.Err() == nil {
		var item json.RawMessage
		offset := dec.InputOffset()
		if err = dec.Decode(&item); err != nil {
			return fmt.Errorf("failed to decode json: %w", err)
		}

		// if expand_event_list_from_field is set, then split the event list
		if j.src.ExpandEventListFromField != "" {
			if numEvents, err := j.splitEventList(j.src.ExpandEventListFromField, item, offset, id); err != nil {
				return err
			} else {
				j.metrics.gcsEventsPerObject.Update(int64(numEvents))
			}
			continue
		} else {
			j.metrics.gcsEventsPerObject.Update(1)
		}

		var parsedData []mapstr.M
		if j.src.ParseJSON {
			parsedData, err = decodeJSON(bytes.NewReader(item))
			if err != nil {
				// since we do not want to stop processing the job here as this is purely cosmetic and optional, we log the error and continue
				j.metrics.errorsTotal.Inc()
				j.metrics.decodeErrorsTotal.Inc()
				j.log.Errorw("job encountered an error during 'ParseJSON' op", "gcs.jobId", id, "error", err)
			}
		}
		evt := j.createEvent(item, parsedData, offset)
		j.publish(evt, !dec.More(), id)
	}
	return nil
}

func (j *job) publish(evt beat.Event, last bool, id string) {
	if last {
		// if this is the last object, then perform a complete state save
		cp, done := j.state.saveForTx(j.object.Name, j.object.Updated, j.metrics)
		if err := j.publisher.Publish(evt, cp); err != nil {
			j.metrics.errorsTotal.Inc()
			j.log.Errorw("job encountered an error while publishing event", "gcs.jobId", id, "error", err)
		}
		done()
		return
	}
	// since we don't update the cursor checkpoint, lack of a lock here is not a problem
	if err := j.publisher.Publish(evt, nil); err != nil {
		j.metrics.errorsTotal.Inc()
		j.log.Errorw("job encountered an error while publishing event", "gcs.jobId", id, "error", err)
	}
}

// splitEventList splits the event list into individual events and publishes them
func (j *job) splitEventList(key string, raw json.RawMessage, offset int64, id string) (int, error) {
	var jsonObject map[string]json.RawMessage
	var eventsPerObject int
	if err := json.Unmarshal(raw, &jsonObject); err != nil {
		return eventsPerObject, fmt.Errorf("job with job id %s encountered an unmarshaling error: %w", id, err)
	}

	raw, found := jsonObject[key]
	if !found {
		return eventsPerObject, fmt.Errorf("expand_event_list_from_field key <%v> is not in event", key)
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	// UseNumber causes the Decoder to unmarshal a number into an interface{} as a Number instead of as a float64.
	dec.UseNumber()

	tok, err := dec.Token()
	if err != nil {
		return eventsPerObject, fmt.Errorf("failed to read JSON token for object: %s, with error: %w", j.object.Name, err)
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '[' {
		return eventsPerObject, fmt.Errorf("expand_event_list_from_field <%v> is not an array", key)
	}

	for dec.More() {
		arrayOffset := dec.InputOffset()

		var item json.RawMessage
		if err := dec.Decode(&item); err != nil {
			return eventsPerObject, fmt.Errorf("failed to decode array item at offset %d: %w", offset+arrayOffset, err)
		}

		data, err := item.MarshalJSON()
		if err != nil {
			return eventsPerObject, fmt.Errorf("job with job id %s encountered a marshaling error: %w", id, err)
		}
		evt := j.createEvent(data, nil, offset+arrayOffset)

		if !dec.More() {
			// if this is the last object, then perform a complete state save
			cp, done := j.state.saveForTx(j.object.Name, j.object.Updated, j.metrics)
			if err := j.publisher.Publish(evt, cp); err != nil {
				j.metrics.errorsTotal.Inc()
				j.log.Errorw("job encountered an error while publishing event", "gcs.jobId", id, "error", err)
			}
			done()
		} else {
			// since we don't update the cursor checkpoint, lack of a lock here is not a problem
			if err := j.publisher.Publish(evt, nil); err != nil {
				j.metrics.errorsTotal.Inc()
				j.log.Errorw("job encountered an error while publishing event", "gcs.jobId", id, "error", err)
			}
		}
		eventsPerObject++
	}

	return eventsPerObject, nil
}

// addGzipDecoderIfNeeded determines whether the given stream of bytes (encapsulated in a buffered reader)
// represents gzipped content or not and adds gzipped decoder if needed. A bufio.Reader is used
// so the function can peek into the byte  stream without consuming it. This makes it convenient for
// code executed after this function call to consume the stream if it wants.
func (j *job) addGzipDecoderIfNeeded(reader *bufio.Reader) (io.Reader, error) {
	isStreamGzipped := false
	// check if stream is gziped or not
	buf, err := reader.Peek(3)
	if err != nil {
		if errors.Is(err, io.EOF) {
			err = nil
		}
		return reader, err
	}

	// gzip magic number (1f 8b) and the compression method (08 for DEFLATE).
	isStreamGzipped = bytes.Equal(buf, []byte{0x1F, 0x8B, 0x08})

	if !isStreamGzipped {
		return reader, nil
	}

	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}

	return gzReader, nil
}

// evaluateJSON, uses a bufio.NewReader & reader.Peek to evaluate if the
// data stream contains a json array as the root element or not, without
// advancing the reader. If the data stream contains an array as the root
// element, the value of the boolean return type is set to true.
func evaluateJSON(reader *bufio.Reader) (io.Reader, bool, error) {
	eof := false
	for i := 0; ; i++ {
		b, err := reader.Peek((i + 1) * 5)
		if errors.Is(err, io.EOF) {
			eof = true
		}
		startByte := i * 5
		for j := 0; j < len(b[startByte:]); j++ {
			char := b[startByte+j : startByte+j+1]
			switch {
			case bytes.Equal(char, []byte("[")):
				return reader, true, nil
			case bytes.Equal(char, []byte("{")):
				return reader, false, nil
			case unicode.IsSpace(bytes.Runes(char)[0]):
				continue
			default:
				return nil, false, fmt.Errorf("unexpected error: JSON data is malformed")
			}
		}
		if eof {
			return nil, false, fmt.Errorf("unexpected error: JSON data is malformed")
		}
	}
}

func (j *job) createEvent(message []byte, data []mapstr.M, offset int64) beat.Event {
	event := beat.Event{
		Timestamp: time.Now(),
		Fields: mapstr.M{
			"message": string(message), // original stringified data
			"log": mapstr.M{
				"offset": offset,
				"file": mapstr.M{
					"path": j.objectURI,
				},
			},
			"gcs": mapstr.M{
				"storage": mapstr.M{
					"bucket": mapstr.M{
						"name": j.src.BucketName,
					},
					"object": mapstr.M{
						"name":         j.object.Name,
						"content_type": j.object.ContentType,
						"json_data":    data, // objectified data, if parseJSON == true, else its empty array
					},
				},
			},
			// Structs are used here in order to save map allocations
			"cloud": struct {
				Provider string `json:"provider"`
			}{
				Provider: "google cloud",
			},
		},
	}
	event.SetID(objectID(j.hash, offset))
	j.metrics.gcsEventsCreatedTotal.Inc()
	return event
}

func objectID(objectHash string, offset int64) string {
	return fmt.Sprintf("%s-%012d", objectHash, offset)
}
