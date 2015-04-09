// Copyright 2013 Matthew Baird
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/araddon/gou"
	"github.com/bmizerany/assert"
	"github.com/packetbeat/elastigo/api"
	"log"
	"strconv"
	"testing"
	"time"
)

//  go test -bench=".*"
//  go test -bench="Bulk"

var (
	buffers        = make([]*bytes.Buffer, 0)
	totalBytesSent int
	messageSets    int
)

func init() {
	flag.Parse()
	if testing.Verbose() {
		gou.SetupLogging("debug")
	}
}

// take two ints, compare, need to be within 5%
func CloseInt(a, b int) bool {
	c := float64(a) / float64(b)
	if c >= .95 && c <= 1.05 {
		return true
	}
	return false
}

func TestBulkIndexerBasic(t *testing.T) {
	InitTests(true)
	indexer := NewBulkIndexer(3)
	indexer.BulkSender = func(buf *bytes.Buffer) error {
		messageSets += 1
		totalBytesSent += buf.Len()
		buffers = append(buffers, buf)
		//		log.Printf("buffer:%s", string(buf.Bytes()))
		return BulkSend(buf)
	}
	done := make(chan bool)
	indexer.Run(done)

	date := time.Unix(1257894000, 0)
	data := map[string]interface{}{"name": "smurfs", "age": 22, "date": time.Unix(1257894000, 0).Format(time.RFC1123Z)}

	err := indexer.Index("users", "user", "1", "", &date, data, true)

	WaitFor(func() bool {
		return len(buffers) > 0
	}, 5)
	// part of request is url, so lets factor that in
	//totalBytesSent = totalBytesSent - len(*eshost)
	assert.T(t, len(buffers) == 1, fmt.Sprintf("Should have sent one operation but was %d", len(buffers)))
	assert.T(t, BulkErrorCt == 0 && err == nil, fmt.Sprintf("Should not have any errors. BulkErroCt: %v, err:%v", BulkErrorCt, err))
	expectedBytes := 166
	assert.T(t, totalBytesSent == expectedBytes, fmt.Sprintf("Should have sent %v bytes but was %v", expectedBytes, totalBytesSent))

	err = indexer.Index("users", "user", "2", "", nil, data, true)
	<-time.After(time.Millisecond * 10) // we need to wait for doc to hit send channel
	// this will test to ensure that Flush actually catches a doc
	indexer.Flush()
	totalBytesSent = totalBytesSent - len(*eshost)
	assert.T(t, err == nil, fmt.Sprintf("Should have nil error  =%v", err))
	assert.T(t, len(buffers) == 2, fmt.Sprintf("Should have another buffer ct=%d", len(buffers)))

	assert.T(t, BulkErrorCt == 0, fmt.Sprintf("Should not have any errors %d", BulkErrorCt))
	expectedBytes = 282 // with refresh
	assert.T(t, CloseInt(totalBytesSent, expectedBytes), fmt.Sprintf("Should have sent %v bytes but was %v", expectedBytes, totalBytesSent))

	done <- true
}

// currently broken in drone.io
func XXXTestBulkUpdate(t *testing.T) {
	InitTests(true)
	api.Port = "9200"
	indexer := NewBulkIndexer(3)
	indexer.BulkSender = func(buf *bytes.Buffer) error {
		messageSets += 1
		totalBytesSent += buf.Len()
		buffers = append(buffers, buf)
		return BulkSend(buf)
	}
	done := make(chan bool)
	indexer.Run(done)

	date := time.Unix(1257894000, 0)
	user := map[string]interface{}{
		"name": "smurfs", "age": 22, "date": time.Unix(1257894000, 0), "count": 1,
	}

	// Lets make sure the data is in the index ...
	_, err := Index("users", "user", "5", nil, user)

	// script and params
	data := map[string]interface{}{
		"script": "ctx._source.count += 2",
	}
	err = indexer.Update("users", "user", "5", "", &date, data, true)
	// So here's the deal. Flushing does seem to work, you just have to give the
	// channel a moment to recieve the message ...
	//	<- time.After(time.Millisecond * 20)
	//	indexer.Flush()
	done <- true

	WaitFor(func() bool {
		return len(buffers) > 0
	}, 5)

	assert.T(t, BulkErrorCt == 0 && err == nil, fmt.Sprintf("Should not have any errors, bulkErrorCt:%v, err:%v", BulkErrorCt, err))

	response, err := Get("users", "user", "5", nil)
	assert.T(t, err == nil, fmt.Sprintf("Should not have any errors  %v", err))
	newCount := response.Source.(map[string]interface{})["count"]
	assert.T(t, newCount.(float64) == 3,
		fmt.Sprintf("Should have update count: %#v ... %#v", response.Source.(map[string]interface{})["count"], response))
}

func TestBulkSmallBatch(t *testing.T) {
	InitTests(true)

	done := make(chan bool)

	date := time.Unix(1257894000, 0)
	data := map[string]interface{}{"name": "smurfs", "age": 22, "date": time.Unix(1257894000, 0)}

	// Now tests small batches
	indexersm := NewBulkIndexer(1)
	indexersm.BufferDelayMax = 100 * time.Millisecond
	indexersm.BulkMaxDocs = 2
	messageSets = 0
	indexersm.BulkSender = func(buf *bytes.Buffer) error {
		messageSets += 1
		return BulkSend(buf)
	}
	indexersm.Run(done)
	<-time.After(time.Millisecond * 20)

	indexersm.Index("users", "user", "2", "", &date, data, true)
	indexersm.Index("users", "user", "3", "", &date, data, true)
	indexersm.Index("users", "user", "4", "", &date, data, true)
	<-time.After(time.Millisecond * 200)
	//	indexersm.Flush()
	done <- true
	assert.T(t, messageSets == 2, fmt.Sprintf("Should have sent 2 message sets %d", messageSets))

}

func XXXTestBulkErrors(t *testing.T) {
	// lets set a bad port, and hope we get a connection refused error?
	api.Port = "27845"
	defer func() {
		api.Port = "9200"
	}()
	BulkDelaySeconds = 1
	indexer := NewBulkIndexerErrors(10, 1)
	done := make(chan bool)
	indexer.Run(done)
	errorCt := 0
	go func() {
		for i := 0; i < 20; i++ {
			date := time.Unix(1257894000, 0)
			data := map[string]interface{}{"name": "smurfs", "age": 22, "date": time.Unix(1257894000, 0)}
			indexer.Index("users", "user", strconv.Itoa(i), "", &date, data, true)
		}
	}()
	var errBuf *ErrorBuffer
	for errBuf = range indexer.ErrorChannel {
		errorCt++
		break
	}
	if errBuf.Buf.Len() > 0 {
		gou.Debug(errBuf.Err)
	}
	assert.T(t, errorCt > 0, fmt.Sprintf("ErrorCt should be > 0 %d", errorCt))
	done <- true
}

/*
BenchmarkBulkSend	18:33:00 bulk_test.go:131: Sent 1 messages in 0 sets totaling 0 bytes
18:33:00 bulk_test.go:131: Sent 100 messages in 1 sets totaling 145889 bytes
18:33:01 bulk_test.go:131: Sent 10000 messages in 100 sets totaling 14608888 bytes
18:33:05 bulk_test.go:131: Sent 20000 messages in 99 sets totaling 14462790 bytes
   20000	    234526 ns/op

*/
func BenchmarkBulkSend(b *testing.B) {
	InitTests(true)
	b.StartTimer()
	totalBytes := 0
	sets := 0
	GlobalBulkIndexer.BulkSender = func(buf *bytes.Buffer) error {
		totalBytes += buf.Len()
		sets += 1
		//log.Println("got bulk")
		return BulkSend(buf)
	}
	for i := 0; i < b.N; i++ {
		about := make([]byte, 1000)
		rand.Read(about)
		data := map[string]interface{}{"name": "smurfs", "age": 22, "date": time.Unix(1257894000, 0), "about": about}
		IndexBulk("users", "user", strconv.Itoa(i), nil, data, true)
	}
	log.Printf("Sent %d messages in %d sets totaling %d bytes \n", b.N, sets, totalBytes)
	if BulkErrorCt != 0 {
		b.Fail()
	}
}

/*
TODO:  this should be faster than above

BenchmarkBulkSendBytes	18:33:05 bulk_test.go:169: Sent 1 messages in 0 sets totaling 0 bytes
18:33:05 bulk_test.go:169: Sent 100 messages in 2 sets totaling 292299 bytes
18:33:09 bulk_test.go:169: Sent 10000 messages in 99 sets totaling 14473800 bytes
   10000	    373529 ns/op

*/
func BenchmarkBulkSendBytes(b *testing.B) {
	InitTests(true)
	about := make([]byte, 1000)
	rand.Read(about)
	data := map[string]interface{}{"name": "smurfs", "age": 22, "date": time.Unix(1257894000, 0), "about": about}
	body, _ := json.Marshal(data)
	b.StartTimer()
	totalBytes := 0
	sets := 0
	GlobalBulkIndexer.BulkSender = func(buf *bytes.Buffer) error {
		totalBytes += buf.Len()
		sets += 1
		return BulkSend(buf)
	}
	for i := 0; i < b.N; i++ {
		IndexBulk("users", "user", strconv.Itoa(i), nil, body, true)
	}
	log.Printf("Sent %d messages in %d sets totaling %d bytes \n", b.N, sets, totalBytes)
	if BulkErrorCt != 0 {
		b.Fail()
	}
}
