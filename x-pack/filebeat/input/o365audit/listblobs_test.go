// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package o365audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/x-pack/filebeat/input/o365audit/poll"
)

const contentType = "Audit.AzureActiveDirectory"

var now = time.Now().UTC()

type blob struct {
	Created    time.Time `json:"contentCreated"`
	Expiration time.Time `json:"contentExpiration"`
	Id         string    `json:"contentId"`
	Type       string    `json:"contentType"`
	Uri        string    `json:"contentUri"`
}

func idDate(d time.Time) string {
	return strings.ReplaceAll(d.Format("20060102150405.999999999"), ".", "")
}

func makeBlob(c time.Time, path string) blob {
	created := c.UTC()
	id := fmt.Sprintf("%s$%s$%s$%s$emea0026",
		idDate(created),
		idDate(created.Add(time.Hour)),
		strings.ReplaceAll(strings.ToLower(contentType), ".", "_"),
		strings.ReplaceAll(contentType, ".", "_"))
	return blob{
		Created:    created,
		Expiration: created.Add(time.Hour * 24 * 7),
		Id:         id,
		Type:       contentType,
		Uri:        "https://test.localhost/" + path,
	}
}

type fakePoll struct {
	queue []poll.Transaction
}

func (f *fakePoll) RenewToken() error {
	return nil
}

func (f *fakePoll) Enqueue(item poll.Transaction) error {
	f.queue = append(f.queue, item)
	return nil
}

func (f *fakePoll) PagedSearchQuery(t testing.TB, lb poll.Transaction, db []blob) (urls []string, next poll.Transaction) {
	const pageSize = 3
	n := len(db)
	var from, to int
	switch v := lb.(type) {
	case listBlob:
		from = 0
	case paginator:
		req, err := autorest.Prepare(&http.Request{}, v.RequestDecorators()...)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		nextArray, ok := req.URL.Query()["nextPage"]
		if !assert.True(t, ok) || len(nextArray) != 1 {
			t.Fatal("nextPage param is missing in pager query")
		}
		from, err = strconv.Atoi(nextArray[0])
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
	}
	if to = from + pageSize; to > n {
		to = n
	}
	result := db[from:to]
	nextUrl := ""
	if to < n {
		nextUrl = fmt.Sprintf("http://localhost.test/something?nextPage=%d", to)
	}
	return f.deliverResult(t, lb, result, nextUrl)
}

func (f *fakePoll) deliverResult(t testing.TB, pl poll.Transaction, msg interface{}, nextUrl string) (urls []string, next poll.Transaction) {
	js, err := json.Marshal(msg)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	response := &http.Response{
		StatusCode:    200,
		Body:          ioutil.NopCloser(bytes.NewReader(js)),
		ContentLength: int64(len(js)),
	}
	if nextUrl != "" {
		response.Header = http.Header{
			"NextPageUri": []string{nextUrl},
		}
	}
	return f.finishQuery(t, pl, response)
}

func (f *fakePoll) SearchQuery(t testing.TB, lb listBlob, db []blob) (urls []string, next poll.Transaction) {
	t.Log("Query start:", now.Sub(lb.startTime), "end:", now.Sub(lb.endTime))
	lowerBound := sort.Search(len(db), func(i int) bool {
		return !db[i].Created.Before(lb.startTime)
	})
	upperBound := sort.Search(len(db), func(i int) bool {
		return !db[i].Created.Before(lb.endTime)
	})
	result := db[lowerBound:upperBound]
	return f.deliverResult(t, lb, result, "")
}

func (f *fakePoll) finishQuery(t testing.TB, pl poll.Transaction, resp *http.Response) (urls []string, next poll.Transaction) {
	for _, a := range pl.OnResponse(resp) {
		if err := a(f); !assert.NoError(t, err) {
			t.Fatal(err)
		}
	}
	if n := len(f.queue); n > 0 {
		urls = make([]string, n-1)
		for i := 0; i < n-1; i++ {
			req, err := autorest.Prepare(&http.Request{}, f.queue[i].RequestDecorators()...)
			if !assert.NoError(t, err) {
				t.Fatal(err)
			}
			urls[i] = req.URL.Path[1:]
		}
		next = f.queue[n-1]
	}
	f.queue = nil
	return urls, next
}

func (f *fakePoll) subscriptionError(t testing.TB, lb listBlob) (subscribe, listBlob) {
	t.Log("Query start:", now.Sub(lb.startTime), "end:", now.Sub(lb.endTime))
	var apiErr apiError
	apiErr.Error.Code = "AF20022"
	apiErr.Error.Message = "No subscription found for the specified content type"
	js, err := json.Marshal(apiErr)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	t.Log(string(js))
	resp := &http.Response{
		StatusCode: 400,
		Body:       ioutil.NopCloser(bytes.NewReader(js)),
	}
	for _, a := range lb.OnResponse(resp) {
		if err := a(f); !assert.NoError(t, err) {
			t.Fatal(err)
		}
	}
	if !assert.Len(t, f.queue, 2) {
		t.Fatal("need 2 actions")
	}
	if !assert.IsType(t, subscribe{}, f.queue[0]) {
		t.Fatal("expected type not found")
	}
	if !assert.IsType(t, lb, f.queue[1]) {
		t.Fatal("expected type not found")
	}
	return f.queue[0].(subscribe), f.queue[1].(listBlob)
}

func testConfig() apiEnvironment {
	logp.TestingSetup()
	config := defaultConfig()
	return apiEnvironment{
		Config: config.API,
		Logger: logp.NewLogger(pluginName + " test"),
		Clock: func() time.Time {
			return now
		},
	}
}

func TestListBlob(t *testing.T) {
	ctx := testConfig()

	db := []blob{
		// 7d+ ago
		makeBlob(now.Add(-time.Hour*(1+24*7)), "expired"),
		// [7,6d) ago
		makeBlob(now.Add(-time.Hour*(8+24*6)), "day1_1"),
		makeBlob(now.Add(-time.Hour*(3+24*6)), "day1_2"),
		// [6d,5d) ago
		makeBlob(now.Add(-time.Hour*(3+24*5)), "day2_1"),

		// [5d-4d) ago
		makeBlob(now.Add(-time.Hour*(24*5)), "day3_1_limit"),
		makeBlob(now.Add(-time.Hour*(23+24*4)), "day3_2"),
		// Yesterday
		makeBlob(now.Add(-time.Hour*(12+24*1)), "day6"),
		// Today
		makeBlob(now.Add(-time.Hour*12), "today_1"),
		makeBlob(now.Add(-time.Hour*7), "today_2"),
	}
	ctx.TenantID = "1234"
	ctx.ContentType = contentType
	lb := makeListBlob(checkpoint{}, ctx)
	var f fakePoll
	// 6 days ago
	blobs, next := f.SearchQuery(t, lb, db)
	assert.Equal(t, []string{"day1_1", "day1_2"}, blobs)
	assert.IsType(t, listBlob{}, next)
	// 5 days ago
	blobs, next = f.SearchQuery(t, next.(listBlob), db)
	assert.Equal(t, []string{"day2_1"}, blobs)

	// 4 days ago
	blobs, next = f.SearchQuery(t, next.(listBlob), db)
	assert.Equal(t, []string{"day3_1_limit", "day3_2"}, blobs)

	// 3 days ago
	blobs, next = f.SearchQuery(t, next.(listBlob), db)
	assert.Empty(t, blobs)

	// 2 days ago
	blobs, next = f.SearchQuery(t, next.(listBlob), db)
	assert.Empty(t, blobs)

	// Yesterday
	blobs, next = f.SearchQuery(t, next.(listBlob), db)
	assert.Equal(t, []string{"day6"}, blobs)

	// Today
	blobs, next = f.SearchQuery(t, next.(listBlob), db)
	assert.Equal(t, []string{"today_1", "today_2"}, blobs)

	// Query for new data
	blobs, next = f.SearchQuery(t, next.(listBlob), db)
	assert.Empty(t, blobs)

	// New blob
	db = append(db, makeBlob(now.Add(-time.Hour*5), "live_1"))

	blobs, next = f.SearchQuery(t, next.(listBlob), db)
	assert.Equal(t, []string{"live_1"}, blobs)

	blobs, next = f.SearchQuery(t, next.(listBlob), db)
	assert.Empty(t, blobs)

	// Two new blobs
	db = append(db, makeBlob(now.Add(-time.Hour*5+time.Second), "live_2"))
	db = append(db, makeBlob(now.Add(-time.Hour*5+2*time.Second), "live_3"))

	blobs, next = f.SearchQuery(t, next.(listBlob), db)
	assert.Equal(t, []string{"live_2", "live_3"}, blobs)

	blobs, next = f.SearchQuery(t, next.(listBlob), db)
	assert.Empty(t, blobs)

	blobs, next = f.SearchQuery(t, next.(listBlob), db)
	assert.Empty(t, blobs)

	// Two more blobs with the same timestamp.
	// I don't even know if this is possible, but assuming that in this case
	// they will have a different ID because the ID uses the timestamp up to a
	// nanosecond precision while the date only has millisecond-precision.
	db = append(db, makeBlob(now.Add(-time.Hour*3+time.Nanosecond), "live_4a"))
	db = append(db, makeBlob(now.Add(-time.Hour*3+2*time.Nanosecond), "live_4b"))
	db = append(db, makeBlob(now.Add(-time.Hour*3+3*time.Nanosecond), "live_4c"))

	blobs, next = f.SearchQuery(t, next.(listBlob), db)
	assert.Equal(t, []string{"live_4a", "live_4b", "live_4c"}, blobs)

	blobs, _ = f.SearchQuery(t, next.(listBlob), db)
	assert.Empty(t, blobs)
}

func TestSubscriptionStart(t *testing.T) {
	logp.TestingSetup()
	log := logp.L()
	ctx := apiEnvironment{
		ContentType: contentType,
		TenantID:    "1234",
		Logger:      log,
		Clock: func() time.Time {
			return now
		},
	}
	ctx.TenantID = "1234"
	ctx.ContentType = contentType
	lb := makeListBlob(checkpoint{}, ctx)
	var f fakePoll
	s, l := f.subscriptionError(t, lb)
	assert.Equal(t, lb.cursor, l.cursor)
	assert.Equal(t, lb.endTime, l.endTime)
	assert.Equal(t, lb.startTime, l.startTime)
	assert.Equal(t, lb.delay, l.delay)
	assert.Equal(t, lb.cursor, l.cursor)
	assert.Equal(t, lb.env.TenantID, l.env.TenantID)
	assert.Equal(t, lb.env.ContentType, l.env.ContentType)
	assert.Equal(t, lb.env.Logger, l.env.Logger)
	assert.Equal(t, contentType, s.ContentType)
	assert.Equal(t, "1234", s.TenantID)
}

func TestPagination(t *testing.T) {
	ctx := testConfig()
	db := []blob{
		makeBlob(now.Add(-time.Hour*47+1*time.Nanosecond), "e1"),
		makeBlob(now.Add(-time.Hour*47+2*time.Nanosecond), "e2"),
		makeBlob(now.Add(-time.Hour*47+3*time.Nanosecond), "e3"),
		makeBlob(now.Add(-time.Hour*47+4*time.Nanosecond), "e4"),
		makeBlob(now.Add(-time.Hour*47+5*time.Nanosecond), "e5"),
		makeBlob(now.Add(-time.Hour*47+6*time.Nanosecond), "e6"),
		makeBlob(now.Add(-time.Hour*47+7*time.Nanosecond), "e7"),
		makeBlob(now.Add(-time.Hour*47+8*time.Nanosecond), "e8"),
	}
	ctx.TenantID = "1234"
	ctx.ContentType = contentType
	lb := makeListBlob(checkpoint{Timestamp: now.Add(-time.Hour * 48)}, ctx)
	var f fakePoll
	// 6 days ago
	blobs, next := f.PagedSearchQuery(t, lb, db)
	assert.Equal(t, []string{"e1", "e2", "e3"}, blobs)
	assert.IsType(t, paginator{}, next)

	blobs, next = f.PagedSearchQuery(t, next, db)
	assert.Equal(t, []string{"e4", "e5", "e6"}, blobs)
	assert.IsType(t, paginator{}, next)

	blobs, next = f.PagedSearchQuery(t, next, db)
	assert.Equal(t, []string{"e7", "e8"}, blobs)
	nextlb, ok := next.(listBlob)
	if !assert.True(t, ok) {
		t.Fatal("bad type after pagination")
	}
	assert.Equal(t, lb.endTime, nextlb.startTime)
	assert.True(t, lb.endTime.Before(nextlb.endTime))
}

func mkTime(t testing.TB, str string) time.Time {
	tm, err := time.Parse(apiDateFormat, str)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	return tm
}

func TestAdvance(t *testing.T) {
	start := mkTime(t, "2020-02-01T15:00:00")
	ev1 := mkTime(t, "2020-02-02T12:00:00")
	now1 := mkTime(t, "2020-02-03T00:00:00")
	ev2 := mkTime(t, "2020-02-03T12:00:00")
	now2 := mkTime(t, "2020-02-04T00:00:00")
	now3 := mkTime(t, "2020-02-06T00:00:00")
	db := []blob{
		makeBlob(ev1, "e1"),
		makeBlob(ev2, "e2"),
	}
	now := &now1
	ctx := testConfig()
	ctx.Clock = func() time.Time {
		return *now
	}
	ctx.TenantID = "tenant"
	ctx.ContentType = contentType
	lb := makeListBlob(checkpoint{Timestamp: start}, ctx)
	assert.Equal(t, start, lb.startTime)
	assert.Equal(t, start.Add(time.Hour*24), lb.endTime)
	assert.True(t, lb.endTime.Before(now1))
	var f fakePoll
	blobs, next := f.SearchQuery(t, lb, db)
	assert.Equal(t, []string{"e1"}, blobs)
	assert.IsType(t, listBlob{}, next)
	lb = next.(listBlob)
	assert.Equal(t, ev1, lb.startTime)
	assert.Equal(t, now1, lb.endTime)

	now = &now2
	blobs, next = f.SearchQuery(t, lb, db)
	assert.Empty(t, blobs)
	assert.IsType(t, listBlob{}, next)
	lb = next.(listBlob)
	assert.Equal(t, now1, lb.startTime)
	assert.Equal(t, now2, lb.endTime)

	blobs, next = f.SearchQuery(t, lb, db)
	assert.Equal(t, []string{"e2"}, blobs)
	assert.IsType(t, listBlob{}, next)
	lb = next.(listBlob)
	assert.Equal(t, ev1.Add(time.Hour*24), lb.startTime)
	assert.Equal(t, now2, lb.endTime)

	now = &now3
	blobs, next = f.SearchQuery(t, lb, db)
	assert.Empty(t, blobs)
	assert.IsType(t, listBlob{}, next)
	lb = next.(listBlob)
	assert.Equal(t, now2, lb.startTime)
	assert.Equal(t, now2.Add(time.Hour*24), lb.endTime)

	blobs, next = f.SearchQuery(t, lb, db)
	assert.Empty(t, blobs)
	assert.IsType(t, listBlob{}, next)
	lb = next.(listBlob)
	assert.Equal(t, now2.Add(time.Hour*24), lb.startTime)
	assert.Equal(t, now3, lb.endTime)
}
