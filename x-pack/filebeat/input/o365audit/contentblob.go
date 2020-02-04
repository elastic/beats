// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package o365audit

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/x-pack/filebeat/input/o365audit/poll"
)

// contentBlob is a poll.Transaction that processes "content blobs":
// aggregations of audit event objects returned by the API.
type contentBlob struct {
	env     apiEnvironment
	id, url string
	// cursor is used to ACK the resulting events.
	cursor cursor
	// skipLines is used when resuming from a saved cursor so that already
	// acknowledged objects are not duplicated.
	skipLines int
}

// String return a printable representation of this transaction.
func (c contentBlob) String() string {
	return fmt.Sprintf("content blob url:%s id:%s", c.url, c.id)
}

// RequestDecorators returns the decorators used to perform a request.
func (c contentBlob) RequestDecorators() []autorest.PrepareDecorator {
	return []autorest.PrepareDecorator{
		autorest.WithBaseURL(c.url),
	}
}

// Delay returns the delay to perform this request.
func (c contentBlob) Delay() time.Duration {
	return 0
}

// OnResponse parses the response for a content blob.
func (c contentBlob) OnResponse(response *http.Response) (actions []poll.Action) {
	if response.StatusCode != 200 {
		// TODO:
		return append(actions, poll.Terminate(
			fmt.Errorf("operation %s returned HTTP code %d %s",
				c, response.StatusCode, response.Status)))
	}
	var js []common.MapStr
	if err := readJSONBody(response, &js); err != nil {
		return append(actions, poll.Terminate(errors.Wrap(err, "reading body failed")))
	}
	for idx, entry := range js {
		id, _ := getString(entry, "Id")
		ts, _ := getString(entry, "CreationTime")
		c.env.Logger.Debugf(" > event %d: created:%s id:%s for %s", idx+1, ts, id, c.cursor)
	}
	if len(js) > c.skipLines {
		for _, entry := range js[:c.skipLines] {
			id, _ := getString(entry, "Id")
			c.env.Logger.Debugf("Skipping event %s [%s] for %s", c.cursor, id, c.id)
		}
		for _, entry := range js[c.skipLines:] {
			c.cursor = c.cursor.ForNextLine()
			c.env.Logger.Debugf("Reporting event %s for %s", c.cursor, c.id)
			actions = append(actions, c.env.Report(entry, c.cursor))
		}
		c.skipLines = 0
	} else {
		for _, entry := range js {
			id, _ := getString(entry, "Id")
			c.env.Logger.Debugf("Skipping event all %s [%s] for %s", c.cursor, id, c.id)
		}

		c.skipLines -= len(js)
	}
	// The API only documents the use of NextPageUri header for list requests
	// but one can't be too careful.
	if url, found := getNextPage(response); found {
		return append(actions, poll.Fetch(newPager(url, c)))
	}

	return actions
}

// ContentBlob creates a new contentBlob.
func ContentBlob(url string, cursor cursor, env apiEnvironment) contentBlob {
	return contentBlob{
		url:    url,
		env:    env,
		cursor: cursor,
	}
}

// WithID configures a content blob with the given origin ID.
func (c contentBlob) WithID(id string) contentBlob {
	c.id = id
	return c
}

// WithSkipLines configures a content blob with the number of objects to skip.
func (c contentBlob) WithSkipLines(nlines int) contentBlob {
	c.skipLines = nlines
	return c
}
