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
	"fmt"
	"io"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const jobErrString = "job with jobId %s encountered an error: %w"

type job struct {
	client       *azblob.BlobClient
	blob         *azblob.BlobItemInternal
	blobURL      string
	hash         string
	offset       int64
	isCompressed bool
	state        *state
	src          *Source
	publisher    cursor.Publisher
	log          *logp.Logger
}

// newJob, returns an instance of a job, which is a unit of work that can be assigned to a go routine
func newJob(client *azblob.BlobClient, blob *azblob.BlobItemInternal, blobURL string,
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
func azureObjectHash(src *Source, blob *azblob.BlobItemInternal) string {
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
		j.state.save(*j.blob.Name, *j.blob.Properties.LastModified)
		if err := j.publisher.Publish(event, j.state.checkpoint()); err != nil {
			j.log.Errorf(jobErrString, id, err)
		}
	}
}

func (j *job) name() string {
	return *j.blob.Name
}

func (j *job) timestamp() *time.Time {
	return j.blob.Properties.LastModified
}

func (j *job) processAndPublishData(ctx context.Context, id string) error {
	var err error
	downloadOptions := &azblob.BlobDownloadOptions{}
	if !j.isCompressed {
		downloadOptions.Offset = &j.offset
	}

	get, err := j.client.Download(ctx, downloadOptions)
	if err != nil {
		return fmt.Errorf("failed to download data from blob with error: %w", err)
	}

	reader := get.Body(&azblob.RetryReaderOptions{})
	defer func() {
		err = reader.Close()
		if err != nil {
			err = fmt.Errorf("failed to close blob reader with error: %w", err)
		}
	}()

	updatedReader, err := j.addGzipDecoderIfNeeded(reader)
	err = j.readJsonAndPublish(ctx, updatedReader, id)
	if err != nil {
		return fmt.Errorf("failed to read data from blob with error: %w", err)
	}

	return err
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

func (j *job) readJsonAndPublish(ctx context.Context, r io.Reader, id string) error {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	var offset int64
	var relativeOffset int64
	// uncompressed files use the client to directly set the offset, this
	// in turn causes the offset to reset to 0 for the new stream, hence why
	// we need to keep relative offsets to keep track of the actual offset
	if !j.isCompressed {
		relativeOffset = j.offset
	}
	for dec.More() && ctx.Err() == nil {
		var item json.RawMessage
		offset = dec.InputOffset()
		if err := dec.Decode(&item); err != nil {
			return fmt.Errorf("failed to decode json: %w", err)
		}
		// manually seek offset only if file is compressed
		if j.isCompressed && offset < j.offset {
			continue
		}

		data, err := item.MarshalJSON()
		if err != nil {
			return err
		}
		evt := j.createEvent(string(data), offset+relativeOffset)
		// updates the offset after reading the file
		// this avoids duplicates for the last read when resuming operation
		offset = dec.InputOffset()
		if !dec.More() {
			// if this is the last object, then peform a complete state save
			j.state.save(*j.blob.Name, *j.blob.Properties.LastModified)
		} else {
			// partially saves read state using offset
			j.state.savePartial(*j.blob.Name, offset+relativeOffset, j.blob.Properties.LastModified)
		}
		if err := j.publisher.Publish(evt, j.state.checkpoint()); err != nil {
			j.log.Errorf(jobErrString, id, err)
		}
	}
	return nil
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
						"name":         j.blob.Name,
						"content_type": j.blob.Properties.ContentType,
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
