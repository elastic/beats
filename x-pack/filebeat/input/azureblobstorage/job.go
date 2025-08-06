// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"bufio"
	"bytes"
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
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/x-pack/libbeat/reader"
	"github.com/elastic/beats/v7/x-pack/libbeat/reader/decoder"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

const jobErrString = "job with jobId %s encountered an error: %v"

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
	// job status reporter
	status status.StatusReporter
	// metrics is used to track the input's metrics
	metrics *inputMetrics
}

// newJob, returns an instance of a job, which is a unit of work that can be assigned to a go routine
func newJob(client *blob.Client, blob *azcontainer.BlobItem, blobURL string,
	state *state, src *Source, publisher cursor.Publisher, stat status.StatusReporter, metrics *inputMetrics, log *logp.Logger,
) *job {
	if metrics == nil {
		// metrics are optional, initialize a stub if not provided
		metrics = newInputMetrics(monitoring.NewRegistry())
	}

	return &job{
		client:    client,
		blob:      blob,
		blobURL:   blobURL,
		hash:      azureObjectHash(src, blob),
		state:     state,
		src:       src,
		publisher: publisher,
		log:       log,
		status:    stat,
		metrics:   metrics,
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
	// metrics & logging
	j.log.Debug("begin abs blob processing.")
	j.metrics.absBlobsRequestedTotal.Inc()
	j.metrics.absBlobsInflight.Inc()
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		j.metrics.absBlobsInflight.Dec()
		j.metrics.absBlobProcessingTime.Update(elapsed.Nanoseconds())
		j.log.Debugw("end abs blob processing.", "elapsed_time_ns", elapsed)
	}()

	if allowedContentTypes[*j.blob.Properties.ContentType] {
		if *j.blob.Properties.ContentType == gzType || (j.blob.Properties.ContentEncoding != nil && *j.blob.Properties.ContentEncoding == encodingGzip) {
			j.isCompressed = true
		}
		err := j.processAndPublishData(ctx, id)
		if err != nil {
			j.metrics.errorsTotal.Inc()
			j.log.Errorf(jobErrString, id, err)
			return
		}
		j.metrics.absBlobsPublishedTotal.Inc()
		if j.blob.Properties.ContentLength != nil {
			// this is an approximate value as there is no guarantee that the content length is not nil
			j.metrics.absBytesProcessedTotal.Add(uint64(*j.blob.Properties.ContentLength))
			j.metrics.absBlobSizeInBytes.Update(*j.blob.Properties.ContentLength)
		}
	} else {
		err := fmt.Errorf("job with jobId %s encountered an error: content-type %s not supported", id, *j.blob.Properties.ContentType)
		j.status.UpdateStatus(status.Degraded, fmt.Sprintf("found unsupported content-type: %s", *j.blob.Properties.ContentType))
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
		err = j.publisher.Publish(event, cp)
		if err != nil {
			j.metrics.errorsTotal.Inc()
			j.log.Errorf(jobErrString, id, err)
			j.status.UpdateStatus(status.Degraded, "failed to publish unsupported content-type event: "+err.Error())
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
		j.status.UpdateStatus(status.Degraded, "failed to create a download stream: "+err.Error())
		return fmt.Errorf("failed to download data from blob with error: %w", err)
	}
	const maxRetries = 3
	reader := get.NewRetryReader(ctx, &azblob.RetryReaderOptions{
		MaxRetries: maxRetries,
	})
	defer func() {
		err = reader.Close()
		if err != nil {
			j.metrics.errorsTotal.Inc()
			j.log.Errorw("failed to close blob reader with error:", "blobName", *j.blob.Name, "error", err)
		}
	}()

	// update the source lag time metric. LastModified time will always be present for a blob.
	j.metrics.sourceLagTime.Update(time.Since(*j.blob.Properties.LastModified).Nanoseconds())

	// calculate number of decode errors
	if err := j.decode(ctx, reader, id); err != nil {
		j.metrics.decodeErrorsTotal.Inc()
		return fmt.Errorf("failed to decode blob: %s, with error: %w", *j.blob.Name, err)
	}

	return nil
}

func (j *job) decode(ctx context.Context, r io.Reader, id string) error {
	r, err := reader.AddGzipDecoderIfNeeded(bufio.NewReader(r))
	if err != nil {
		return fmt.Errorf("failed to add gzip decoder to blob: %s, with error: %w", *j.blob.Name, err)
	}
	dec, err := decoder.NewDecoder(j.src.ReaderConfig.Decoding, r)
	if err != nil {
		return err
	}
	var evtOffset int64
	switch dec := dec.(type) {
	case decoder.Decoder:
		defer dec.Close()

		for dec.Next() {
			msg, err := dec.Decode()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				j.status.UpdateStatus(status.Degraded, err.Error())
				return err
			}
			evt := j.createEvent(string(msg), evtOffset)
			j.publish(evt, !dec.More(), id)
		}

	default:
		err = j.readJsonAndPublish(ctx, r, id)
		if err != nil {
			return fmt.Errorf("failed to read data from blob with error: %w", err)
		}
	}

	return nil
}

func (j *job) readJsonAndPublish(ctx context.Context, r io.Reader, id string) error {
	var err error
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
			err = fmt.Errorf("failed to read JSON token for object: %s, with error: %w", *j.blob.Name, err)
			j.status.UpdateStatus(status.Degraded, err.Error())
			return err
		}
	}

	for dec.More() && ctx.Err() == nil {
		var item json.RawMessage
		offset := dec.InputOffset()
		if err := dec.Decode(&item); err != nil {
			err = fmt.Errorf("failed to decode json: %w", err)
			j.status.UpdateStatus(status.Degraded, err.Error())
			return err
		}

		// if expand_event_list_from_field is set, then split the event list
		if j.src.ExpandEventListFromField != "" {
			if numEvents, err := j.splitEventList(j.src.ExpandEventListFromField, item, offset, id); err != nil {
				return err
			} else {
				j.metrics.absEventsPerBlob.Update(int64(numEvents))
			}
			continue
		} else {
			j.metrics.absEventsPerBlob.Update(1)
		}

		data, err := item.MarshalJSON()
		if err != nil {
			return err
		}
		evt := j.createEvent(string(data), offset)
		j.publish(evt, !dec.More(), id)
	}
	return nil
}

func (j *job) publish(evt beat.Event, last bool, id string) {
	if last {
		// if this is the last object, then perform a complete state save
		cp, done := j.state.saveForTx(*j.blob.Name, *j.blob.Properties.LastModified)
		err := j.publisher.Publish(evt, cp)
		if err != nil {
			j.metrics.errorsTotal.Inc()
			j.status.UpdateStatus(status.Degraded, "failed to publish event: "+err.Error())
			j.log.Errorf(jobErrString, id, err)
		} else {
			j.status.UpdateStatus(status.Running, "")
		}
		done()
		return
	}
	// since we don't update the cursor checkpoint, lack of a lock here should be fine
	err := j.publisher.Publish(evt, nil)
	if err != nil {
		j.metrics.errorsTotal.Inc()
		j.status.UpdateStatus(status.Degraded, "failed to publish event: "+err.Error())
		j.log.Errorf(jobErrString, id, err)
	} else {
		j.status.UpdateStatus(status.Running, "")
	}
}

// splitEventList splits the event list into individual events and publishes them
func (j *job) splitEventList(key string, raw json.RawMessage, offset int64, id string) (int, error) {
	var jsonObject map[string]json.RawMessage
	var eventsPerObject int
	if err := json.Unmarshal(raw, &jsonObject); err != nil {
		j.status.UpdateStatus(status.Degraded, "failed to unmarshal JSON: "+err.Error())
		return eventsPerObject, err
	}

	raw, found := jsonObject[key]
	if !found {
		err := fmt.Errorf("expand_event_list_from_field key <%v> is not in event", key)
		j.status.UpdateStatus(status.Degraded, "possible configuration issue: "+err.Error())
		return eventsPerObject, err
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	tok, err := dec.Token()
	if err != nil {
		j.status.UpdateStatus(status.Degraded, "failed to unmarshal JSON: "+err.Error())
		return eventsPerObject, err
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '[' {
		err := fmt.Errorf("expand_event_list_from_field <%v> is not an array", key)
		j.status.UpdateStatus(status.Degraded, "possible configuration issue: "+err.Error())
		return eventsPerObject, err
	}

	for dec.More() {
		arrayOffset := dec.InputOffset()

		var item json.RawMessage
		if err := dec.Decode(&item); err != nil {
			j.status.UpdateStatus(status.Degraded, "failed to unmarshal JSON: "+err.Error())
			return eventsPerObject, fmt.Errorf("failed to decode array item at offset %d: %w", offset+arrayOffset, err)
		}

		data, err := item.MarshalJSON()
		if err != nil {
			j.status.UpdateStatus(status.Degraded, "failed to re-marshal JSON: "+err.Error())
			return eventsPerObject, err
		}
		evt := j.createEvent(string(data), offset+arrayOffset)

		if !dec.More() {
			// if this is the last object, then save checkpoint
			cp, done := j.state.saveForTx(*j.blob.Name, *j.blob.Properties.LastModified)
			err := j.publisher.Publish(evt, cp)
			if err != nil {
				j.metrics.errorsTotal.Inc()
				j.log.Errorf(jobErrString, id, err)
				j.status.UpdateStatus(status.Degraded, "failed to publish event: "+err.Error())
			} else {
				j.status.UpdateStatus(status.Running, "")
			}
			done()
		} else {
			// since we don't update the cursor checkpoint, lack of a lock here should be fine
			err := j.publisher.Publish(evt, nil)
			if err != nil {
				j.metrics.errorsTotal.Inc()
				j.log.Errorf(jobErrString, id, err)
				j.status.UpdateStatus(status.Degraded, "failed to publish event: "+err.Error())
			} else {
				j.status.UpdateStatus(status.Running, "")
			}
		}
		eventsPerObject++
	}

	return eventsPerObject, nil
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
	j.metrics.absEventsCreatedTotal.Inc()
	return event
}

func objectID(objectHash string, offset int64) string {
	return fmt.Sprintf("%s-%012d", objectHash, offset)
}
