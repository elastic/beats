// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

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

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	azcontainer "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const jobErrString = "job with jobId %s encountered an error: %w"

type job struct {
	// client is an azure blob handle
	client *blob.Client
	// blob is an azure blob item handle
	blob *azcontainer.BlobItem
	// azure blob url for the resource
	blobURL string
	// object hash, used in setting event id
	hash string
	// flag to denote if object is gzip compressed or not
	isCompressed bool
	// flag to denote if object's root element is of an array type
	isRootArray bool
	// blob state
	state *state
	// container source struct used for storing container related data
	src *Source
	// publisher is used to publish a beat event to the output stream
	publisher cursor.Publisher
	// custom logger
	log *logp.Logger
}

// newJob, returns an instance of a job, which is a unit of work that can be assigned to a go routine
func newJob(client *blob.Client, blob *azcontainer.BlobItem, blobURL string,
	state *state, src *Source, publisher cursor.Publisher, log *logp.Logger,
) *job {
	return &job{
		client:    client,
		blob:      blob,
		blobURL:   blobURL,
		hash:      azureObjectHash(src, blob),
		state:     state,
		src:       src,
		publisher: publisher,
		log:       log,
	}
}

// azureObjectHash returns a short sha256 hash of the container name + blob name.
func azureObjectHash(src *Source, blob *azcontainer.BlobItem) string {
	h := sha256.New()
	h.Write([]byte(src.ContainerName))
	h.Write([]byte((*blob.Name)))
	prefix := hex.EncodeToString(h.Sum(nil))
	return prefix[:10]
}

func (j *job) do(ctx context.Context, id string) {
	var fields mapstr.M
	if allowedContentTypes[*j.blob.Properties.ContentType] {
		if *j.blob.Properties.ContentType == gzType || (j.blob.Properties.ContentEncoding != nil && *j.blob.Properties.ContentEncoding == encodingGzip) {
			j.isCompressed = true
		}
		err := j.processAndPublishData(ctx, id)
		if err != nil {
			j.log.Errorf(jobErrString, id, err)
			return
		}

	} else {
		err := fmt.Errorf("job with jobId %s encountered an error: content-type %s not supported", id, *j.blob.Properties.ContentType)
		fields = mapstr.M{
			"message": err.Error(),
		}
		event := beat.Event{
			Timestamp: time.Now(),
			Fields:    fields,
		}
		event.SetID(objectID(j.hash, 0))
		// locks while data is being saved to avoid concurrent map read/writes
		cp, done := j.state.saveForTx(*j.blob.Name, *j.blob.Properties.LastModified)
		if err := j.publisher.Publish(event, cp); err != nil {
			j.log.Errorf(jobErrString, id, err)
		}
		// unlocks after data is saved
		done()
	}
}

func (j *job) name() string {
	return *j.blob.Name
}

func (j *job) timestamp() *time.Time {
	return j.blob.Properties.LastModified
}

func (j *job) processAndPublishData(ctx context.Context, id string) error {
	get, err := j.client.DownloadStream(ctx, &blob.DownloadStreamOptions{})
	if err != nil {
		return fmt.Errorf("failed to download data from blob with error: %w", err)
	}
	const maxRetries = 3
	reader := get.NewRetryReader(ctx, &azblob.RetryReaderOptions{
		MaxRetries: maxRetries,
	})
	defer func() {
		err = reader.Close()
		if err != nil {
			err = fmt.Errorf("failed to close blob reader with error: %w", err)
		}
	}()

	err = j.readJsonAndPublish(ctx, reader, id)
	if err != nil {
		return fmt.Errorf("failed to read data from blob with error: %w", err)
	}

	return err
}

func (j *job) readJsonAndPublish(ctx context.Context, r io.Reader, id string) error {
	r, err := j.addGzipDecoderIfNeeded(bufio.NewReader(r))
	if err != nil {
		return fmt.Errorf("failed to add gzip decoder to blob: %s, with error: %w", *j.blob.Name, err)
	}

	// checks if the root element is an array or not
	r, j.isRootArray, err = evaluateJSON(bufio.NewReader(r))
	if err != nil {
		return fmt.Errorf("failed to evaluate json for blob: %s, with error: %w", *j.blob.Name, err)
	}

	dec := json.NewDecoder(r)
	dec.UseNumber()
	// If array is present at root then read json token and advance decoder
	if j.isRootArray {
		_, err := dec.Token()
		if err != nil {
			return fmt.Errorf("failed to read JSON token for object: %s, with error: %w", *j.blob.Name, err)
		}
	}

	for dec.More() && ctx.Err() == nil {
		var item json.RawMessage
		offset := dec.InputOffset()
		if err := dec.Decode(&item); err != nil {
			return fmt.Errorf("failed to decode json: %w", err)
		}
		// if expand_event_list_from_field is set, then split the event list
		if j.src.ExpandEventListFromField != "" {
			if err := j.splitEventList(j.src.ExpandEventListFromField, item, offset, j.hash, id); err != nil {
				return err
			}
			continue
		}

		data, err := item.MarshalJSON()
		if err != nil {
			return err
		}
		evt := j.createEvent(string(data), offset)

		if !dec.More() {
			// if this is the last object, then perform a complete state save
			cp, done := j.state.saveForTx(*j.blob.Name, *j.blob.Properties.LastModified)
			if err := j.publisher.Publish(evt, cp); err != nil {
				j.log.Errorf(jobErrString, id, err)
			}
			done()
		} else {
			// since we don't update the cursor checkpoint, lack of a lock here should be fine
			if err := j.publisher.Publish(evt, nil); err != nil {
				j.log.Errorf(jobErrString, id, err)
			}
		}
	}
	return nil
}

// splitEventList splits the event list into individual events and publishes them
func (j *job) splitEventList(key string, raw json.RawMessage, offset int64, objHash string, id string) error {
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

		data, err := item.MarshalJSON()
		if err != nil {
			return err
		}
		evt := j.createEvent(string(data), offset+arrayOffset)

		if !dec.More() {
			// if this is the last object, then save checkpoint
			cp, done := j.state.saveForTx(*j.blob.Name, *j.blob.Properties.LastModified)
			if err := j.publisher.Publish(evt, cp); err != nil {
				j.log.Errorf(jobErrString, id, err)
			}
			done()
		} else {
			// since we don't update the cursor checkpoint, lack of a lock here should be fine
			if err := j.publisher.Publish(evt, nil); err != nil {
				j.log.Errorf(jobErrString, id, err)
			}
		}
	}

	return nil
}

// addGzipDecoderIfNeeded determines whether the given stream of bytes (encapsulated in a buffered reader)
// represents gzipped content or not and adds gzipped decoder if needed. A buffered reader is used
// so the function can peek into the byte  stream without consuming it. This makes it convenient for
// code executed after this function call to consume the stream if it wants.
func (j *job) addGzipDecoderIfNeeded(body io.Reader) (io.Reader, error) {
	bufReader := bufio.NewReader(body)
	isStreamGzipped := false
	// check if stream is gziped or not
	buf, err := bufReader.Peek(3)
	if err != nil {
		if err == io.EOF {
			err = nil
		}
		return bufReader, err
	}

	// gzip magic number (1f 8b) and the compression method (08 for DEFLATE).
	isStreamGzipped = bytes.Equal(buf, []byte{0x1F, 0x8B, 0x08})

	if !isStreamGzipped {
		return bufReader, nil
	}

	return gzip.NewReader(bufReader)
}

// evaluateJSON uses a bufio.NewReader & reader.Peek to evaluate if the
// data stream contains a json array as the root element or not, without
// advancing the reader. If the data stream contains an array as the root
// element, the value of the boolean return type is set to true.
func evaluateJSON(reader *bufio.Reader) (io.Reader, bool, error) {
	eof := false
	// readSize is the constant value in the incremental read operation, this value is arbitrary
	// but works well for our use case
	const readSize = 5
	for i := 0; ; i++ {
		b, err := reader.Peek((i + 1) * readSize)
		if errors.Is(err, io.EOF) {
			eof = true
		}
		startByte := i * readSize
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

func (j *job) createEvent(message string, offset int64) beat.Event {
	event := beat.Event{
		Timestamp: time.Now(),
		Fields: mapstr.M{
			"message": message,
			"log": mapstr.M{
				"offset": offset,
				"file": mapstr.M{
					"path": j.blobURL,
				},
			},
			"azure": mapstr.M{
				"storage": mapstr.M{
					"container": mapstr.M{
						"name": j.src.ContainerName,
					},
					"blob": mapstr.M{
						"name":         *j.blob.Name,
						"content_type": *j.blob.Properties.ContentType,
					},
				},
			},
			"cloud": mapstr.M{
				"provider": "azure",
			},
		},
	}
	event.SetID(objectID(j.hash, offset))

	return event
}

func objectID(objectHash string, offset int64) string {
	return fmt.Sprintf("%s-%012d", objectHash, offset)
}
