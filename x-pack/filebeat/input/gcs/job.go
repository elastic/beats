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
	// offset value for an object, it points to the location inside the data stream
	// from where we can start processing the object.
	offset int64
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
	// custom logger
	log *logp.Logger
	// flag used to denote if this object has previously failed without being processed at all.
	isFailed bool
}

// newJob, returns an instance of a job, which is a unit of work that can be assigned to a go routine
func newJob(bucket *storage.BucketHandle, object *storage.ObjectAttrs, objectURI string,
	state *state, src *Source, publisher cursor.Publisher, log *logp.Logger, isFailed bool,
) *job {
	return &job{
		bucket:    bucket,
		object:    object,
		objectURI: objectURI,
		hash:      gcsObjectHash(src, object),
		state:     state,
		src:       src,
		publisher: publisher,
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

	if allowedContentTypes[j.object.ContentType] {
		if j.object.ContentType == gzType || j.object.ContentEncoding == encodingGzip {
			j.isCompressed = true
		}
		if result, ok := j.state.cp.IsRootArray[j.object.Name]; ok {
			j.isRootArray = result
		}
		err := j.processAndPublishData(ctx, id)
		if err != nil {
			j.state.updateFailedJobs(j.object.Name)
			j.log.Errorw("job encountered an error", "gcs.jobId", id, "error", err)
			return
		}

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
		cp, done := j.state.saveForTx(j.object.Name, j.object.Updated)
		if err := j.publisher.Publish(event, cp); err != nil {
			j.log.Errorw("job encountered an error", "gcs.jobId", id, "error", err)
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
	var err error
	var offset int64
	ctxWithTimeout, cancel := context.WithTimeout(ctx, j.src.BucketTimeOut)
	defer cancel()
	obj := j.bucket.Object(j.object.Name)
	// if object is compressed or object root element is an array, then we cannot use an
	// offset to read as it will produce an erroneous data stream.
	if !j.isCompressed && !j.isRootArray {
		offset = j.offset
	}
	reader, err := obj.NewRangeReader(ctxWithTimeout, offset, -1)
	if err != nil {
		return fmt.Errorf("failed to open reader for object: %s, with error: %w", j.object.Name, err)
	}
	defer func() {
		err = reader.Close()
		if err != nil {
			j.log.Errorw("failed to close reader for object", "objectName", j.object.Name, "error", err)
		}
	}()

	err = j.readJsonAndPublish(ctx, reader, id)
	if err != nil {
		return fmt.Errorf("failed to read data from object: %s, with error: %w", j.object.Name, err)
	}

	return err
}

func (j *job) readJsonAndPublish(ctx context.Context, r io.Reader, id string) error {
	var err error
	r, err = j.addGzipDecoderIfNeeded(bufio.NewReader(r))
	if err != nil {
		return fmt.Errorf("failed to add gzip decoder to object: %s, with error: %w", j.object.Name, err)
	}

	// if offset == 0, then this is a new stream which has not been processed previously
	if j.offset == 0 {
		r, j.isRootArray, err = evaluateJSON(bufio.NewReader(r))
		if err != nil {
			return fmt.Errorf("failed to evaluate json for object: %s, with error: %w", j.object.Name, err)
		}
		if j.isRootArray {
			j.state.setRootArray(j.object.Name)
		}
	}
	dec := json.NewDecoder(r)
	// If array is present at root then read json token and advance decoder
	if j.isRootArray {
		_, err := dec.Token()
		if err != nil {
			return fmt.Errorf("failed to read JSON token for object: %s, with error: %w", j.object.Name, err)
		}
	}

	var offset, relativeOffset int64
	// uncompressed files use the client to directly set the offset, this
	// in turn causes the offset to reset to 0 for the new stream, hence why
	// we need to keep relative offsets to keep track of the actual offset
	if !j.isCompressed && !j.isRootArray {
		relativeOffset = j.offset
	}
	dec.UseNumber()
	for dec.More() && ctx.Err() == nil {
		var item json.RawMessage
		offset = dec.InputOffset()
		if err = dec.Decode(&item); err != nil {
			return fmt.Errorf("failed to decode json: %w", err)
		}
		// manually seek offset only if file is compressed or if root element is an array
		if (j.isCompressed || j.isRootArray) && offset < j.offset {
			continue
		}
		var parsedData []mapstr.M
		if j.src.ParseJSON {
			parsedData, err = decodeJSON(bytes.NewReader(item))
			if err != nil {
				j.log.Errorw("job encountered an error", "gcs.jobId", id, "error", err)
			}
		}
		evt := j.createEvent(item, parsedData, offset+relativeOffset)
		// updates the offset after reading the file
		// this avoids duplicates for the last read when resuming operation
		offset = dec.InputOffset()
		// locks while data is being saved and published to avoid concurrent map read/writes
		var (
			done func()
			cp   *Checkpoint
		)
		if !dec.More() {
			// if this is the last object, then peform a complete state save
			cp, done = j.state.saveForTx(j.object.Name, j.object.Updated)
		} else {
			// partially saves read state using offset
			cp, done = j.state.savePartialForTx(j.object.Name, offset+relativeOffset)
		}
		if err := j.publisher.Publish(evt, cp); err != nil {
			j.log.Errorw("job encountered an error", "gcs.jobId", id, "error", err)
		}
		// unlocks after data is saved and published
		done()
	}
	return nil
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

	return event
}

func objectID(objectHash string, offset int64) string {
	return fmt.Sprintf("%s-%012d", objectHash, offset)
}
