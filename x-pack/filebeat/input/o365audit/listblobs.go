// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package o365audit

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/x-pack/filebeat/input/o365audit/poll"
)

// listBlob is a poll.Transaction that handles the content/"blobs" list.
type listBlob struct {
	cursor             checkpoint
	startTime, endTime time.Time
	delay              time.Duration
	env                apiEnvironment
}

// makeListBlob creates a new poll.Transaction that lists content starting from
// the given cursor position.
func makeListBlob(cursor checkpoint, env apiEnvironment) listBlob {
	l := listBlob{
		cursor: cursor,
		env:    env,
	}
	return l.adjustTimes(cursor.Timestamp)
}

// WithStartTime allows to alter the startTime of a listBlob. This is necessary
// for requests that are resuming from the cursor position of an existing blob,
// as it has been observed that the server won't return the same blob, but a
// partial one, when queried with the time that this blob was created.
func (l listBlob) WithStartTime(start time.Time) listBlob {
	return l.adjustTimes(start)
}

func (l listBlob) adjustTimes(since time.Time) listBlob {
	now := l.env.Clock()
	// Can't query more than <retention limit> in the past.
	fromLimit := now.Add(-l.env.Config.MaxRetention)
	if since.Before(fromLimit) {
		since = fromLimit
	}

	to := since.Add(l.env.Config.MaxQuerySize)
	// Can't query into the future. Polling for new events every interval.
	var delay time.Duration
	if to.After(now) {
		since = now.Add(-l.env.Config.MaxQuerySize)
		if since.Before(l.cursor.Timestamp) {
			since = l.cursor.Timestamp
		}
		to = now
		delay = l.env.Config.PollInterval
	}
	l.startTime = since.UTC()
	l.endTime = to.UTC()
	l.delay = delay
	return l
}

// Delay returns the delay before executing a transaction.
func (l listBlob) Delay() time.Duration {
	return l.delay
}

// String returns the printable representation of a listBlob.
func (l listBlob) String() string {
	return fmt.Sprintf("list blobs from:%s to:%s", l.startTime, l.endTime)
}

// RequestDecorators returns the decorators used to perform a request.
func (l listBlob) RequestDecorators() []autorest.PrepareDecorator {
	return []autorest.PrepareDecorator{
		autorest.WithBaseURL(l.env.Config.Resource),
		autorest.WithPath("api/v1.0"),
		autorest.WithPath(l.env.TenantID),
		autorest.WithPath("activity/feed/subscriptions/content"),
		autorest.WithQueryParameters(
			map[string]interface{}{
				"contentType": l.env.ContentType,
				"startTime":   l.startTime.Format(apiDateFormat),
				"endTime":     l.endTime.Format(apiDateFormat),
			}),
	}
}

// OnResponse handles the output of a list content request.
func (l listBlob) OnResponse(response *http.Response) (actions []poll.Action) {
	if response.StatusCode != 200 {
		return l.handleError(response)
	}

	if delta := getServerTimeDelta(response); l.env.Config.AdjustClockWarn && !inRange(delta, l.env.Config.AdjustClockMinDifference) {
		l.env.Logger.Warnf("Server clock is offset by %v: Check system clock to avoid event loss.", delta)
	}

	var list []content
	if err := readJSONBody(response, &list); err != nil {
		return []poll.Action{
			poll.Terminate(err),
		}
	}

	// Sort content by creation date and then by ID.
	sort.Slice(list, func(i, j int) bool {
		return list[i].Created.Before(list[j].Created) || (list[i].Created == list[j].Created && list[i].ID < list[j].ID)
	})

	// Save in the cursor the startTime that was used to obtain this blobs.
	// In case of resuming retrieval using that cursor, it will be necessary to
	// use the same startTime to observe the same blobs. Otherwise there's the
	// risk of observing partial blobs.
	l.cursor = l.cursor.WithStartTime(l.startTime)

	for _, entry := range list {
		// Only fetch blobs that advance the cursor.
		if l.cursor.TryAdvance(entry) {
			l.env.Logger.Debugf("+ fetch blob date:%v id:%s", entry.Created.UTC(), entry.ID)
			actions = append(actions, poll.Fetch(
				ContentBlob(entry.URI, l.cursor, l.env).
					WithID(entry.ID).
					WithSkipLines(l.cursor.Line)))
		} else {
			l.env.Logger.Debugf("- skip blob date:%v id:%s", entry.Created.UTC(), entry.ID)
		}
		if entry.Created.Before(l.startTime) {
			l.env.Logger.Errorf("! Event created before query")
		}
		if entry.Created.After(l.endTime) {
			l.env.Logger.Errorf("! Event created after query")
		}
	}
	// Fetch the next page if a NextPageUri header is found.
	if url, found := getNextPage(response); found {
		return append(actions, poll.Fetch(newPager(url, l)))
	}
	// Otherwise fetch the next time window.
	return append(actions, poll.Fetch(l.Next()))
}

// Next returns a listBlob that will fetch events in future.
func (l listBlob) Next() listBlob {
	return l.adjustTimes(l.endTime)
}

var fatalErrors = map[string]struct{}{
	// Missing parameter: {0}.
	"AF20001": {},
	// Invalid parameter type: {0}. Expected type: {1}
	"AF20002": {},
	// Expiration {0} provided is set to past date and time.
	"AF20003": {},
	// The tenant ID passed in the URL ({0}) does not match the tenant ID passed in the access token ({1}).
	"AF20010": {},
	// Specified tenant ID ({0}) does not exist in the system or has been deleted.
	"AF20011": {},
	// Specified tenant ID ({0}) is incorrectly configured in the system.
	"AF20012": {},
	// The tenant ID passed in the URL ({0}) is not a valid GUID.
	"AF20013": {},
	// The specified content type is not valid.
	"AF20020": {},
	// The webhook endpoint {{0}) could not be validated. {1}
	"AF20021": {},
}

func (l listBlob) handleError(response *http.Response) (actions []poll.Action) {
	var msg apiError
	readJSONBody(response, &msg)
	l.env.Logger.Warnf("Got error %s: %+v", response.Status, msg)
	l.delay = l.env.Config.ErrorRetryInterval

	switch response.StatusCode {
	case 401:
		// Authentication error. Renew oauth token and repeat this op.
		l.delay = l.env.Config.PollInterval
		return []poll.Action{
			poll.RenewToken(),
			poll.Fetch(l),
		}
	case 408, 503:
		// Known errors when the backend is down.
		// Repeat the request without reporting an error.
		return []poll.Action{
			poll.Fetch(l),
		}
	}

	if _, found := fatalErrors[msg.Error.Code]; found {
		return []poll.Action{
			l.env.ReportAPIError(msg),
			poll.Terminate(errors.New(msg.Error.Message)),
		}
	}

	switch msg.Error.Code {
	// AF20022: No subscription found for the specified content type
	// AF20023: The subscription was disabled by [..]
	case "AF20022", "AF20023":
		l.delay = 0
		// Subscribe and retry
		return []poll.Action{
			poll.Fetch(Subscribe(l.env)),
			poll.Fetch(l),
		}
	// AF20030: Start time and end time must both be specified (or both omitted) and must
	// be less than or equal to 24 hours apart, with the start time no more than
	// 7 days in the past.
	// AF20055: (Same).
	case "AF20030", "AF20055":
		// As of writing this, the server fails a request if it's more than
		// retention_time(7d)+1h in the past.
		// On the other hand, requests can be days into the future without error.

		// First check if this is caused by a request close to the max retention
		// period that's been queued for hours because of server being down.
		// Repeat the request with updated times.
		now := l.env.Clock()
		delta := now.Sub(l.startTime)
		if delta > (l.env.Config.MaxRetention + 30*time.Minute) {
			l.delay = l.env.Config.PollInterval
			return []poll.Action{
				poll.Fetch(l.adjustTimes(l.startTime)),
			}
		}

		delta = getServerTimeDelta(response)
		l.env.Logger.Errorf("Server is complaining about query interval. "+
			"This is usually a problem with the local clock and the server's clock "+
			"being out of sync. Time difference with server is %v.", delta)
		if l.env.Config.AdjustClock && !inRange(delta, l.env.Config.AdjustClockMinDifference) {
			l.env.Clock = func() time.Time {
				return time.Now().Add(delta)
			}
			l.env.Logger.Info("Compensating for time difference")
		} else {
			l.env.Logger.Infow("Not adjusting for time offset.",
				"api.adjust_clock", l.env.Config.AdjustClock,
				"api.adjust_clock_min_difference", l.env.Config.AdjustClockMinDifference,
				"difference", delta)
		}
		return []poll.Action{
			poll.Fetch(l.adjustTimes(l.startTime)),
		}

	// Too many requests.
	case "AF429":

	// Internal server error. Retry the request.
	case "AF50000":

	// Invalid nextPage Input: {0}. Can be ignored.
	case "AF20031":

	// AF50005-AF50006: An internal error occurred. Retry the request.
	case "AF50005", "AF50006":
		return append(actions, poll.Fetch(l))
	}

	if msg.Error.Code != "" {
		actions = append(actions, l.env.ReportAPIError(msg))
	}
	return append(actions, poll.Fetch(l))
}

func readJSONBody(response *http.Response, dest interface{}) error {
	defer autorest.Respond(response,
		autorest.ByDiscardingBody(),
		autorest.ByClosing())
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return errors.Wrap(err, "reading body failed")
	}
	if err = json.Unmarshal(body, dest); err != nil {
		return errors.Wrap(err, "decoding json failed")
	}
	return nil
}

func getServerTimeDelta(response *http.Response) time.Duration {
	serverDate, err := httpDateFormats.Parse(response.Header.Get("Date"))
	if err != nil {
		return 0
	}
	return time.Until(serverDate)
}
