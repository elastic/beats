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
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/packetbeat/elastigo/api"
	"io"
	"log"
	"net/http"
	"os"
	"testing"
	"time"
)

/*

usage:

	test -v -host eshost -loaddata

*/

var (
	_                 = os.ModeDir
	bulkStarted       bool
	hasLoadedData     bool
	hasStartedTesting bool
	eshost            *string = flag.String("host", "localhost", "Elasticsearch Server Host Address")
	esport            *string = flag.String("port", "9200", "Elasticsearch Server Port")
	espath            *string = flag.String("path", "", "Elasticsearch HTTP Path. E.g.: /elasticsearch")
	loadData          *bool   = flag.Bool("loaddata", false, "This loads a bunch of test data into elasticsearch for testing")
)

func init() {

}
func InitTests(startIndexer bool) {
	if !hasStartedTesting {
		flag.Parse()
		hasStartedTesting = true
		log.SetFlags(log.Ltime | log.Lshortfile)
		api.Domain = *eshost
		api.Port = *esport
		api.BasePath = *espath
	}
	if startIndexer && !bulkStarted {
		BulkDelaySeconds = 1
		bulkStarted = true
		BulkIndexerGlobalRun(100, make(chan bool))
		if *loadData && !hasLoadedData {
			log.Println("loading test data ")
			hasLoadedData = true
			LoadTestData()
		}
	}
}

// dumb simple assert for testing, printing
//    Assert(len(items) == 9, t, "Should be 9 but was %d", len(items))
func Assert(is bool, t *testing.T, format string, args ...interface{}) {
	if is == false {
		log.Printf(format, args...)
		t.Fail()
	}
}

// Wait for condition (defined by func) to be true, a utility to create a ticker
// checking every 100 ms to see if something (the supplied check func) is done
//
//   WaitFor(func() bool {
//      return ctr.Ct == 0
//   },10)
//
// @timeout (in seconds) is the last arg
func WaitFor(check func() bool, timeoutSecs int) {
	timer := time.NewTicker(100 * time.Millisecond)
	tryct := 0
	for _ = range timer.C {
		if check() {
			timer.Stop()
			break
		}
		if tryct >= timeoutSecs*10 {
			timer.Stop()
			break
		}
		tryct++
	}
}

func TestFake(t *testing.T) {

}

type GithubEvent struct {
	Url     string
	Created time.Time `json:"created_at"`
	Type    string
}

// This loads test data from github archives (~6700 docs)
func LoadTestData() {
	docCt := 0
	errCt := 0
	indexer := NewBulkIndexer(1)
	indexer.BulkSender = func(buf *bytes.Buffer) error {
		//		log.Printf("Sent %d bytes total %d docs sent", buf.Len(), docCt)
		req, err := api.ElasticSearchRequest("POST", "/_bulk", "")
		if err != nil {
			errCt += 1
			log.Fatalf("ERROR: ", err)
			return err
		}
		req.SetBody(buf)
		//		res, err := http.DefaultClient.Do(*(api.Request(req)))
		var response map[string]interface{}
		httpStatusCode, _, err := req.Do(&response)
		if err != nil {
			errCt += 1
			log.Fatalf("ERROR: %v", err)
			return err
		}
		if httpStatusCode != 200 {
			log.Fatalf("Not 200! %d \n", httpStatusCode)
		}
		return err
	}
	done := make(chan bool)
	indexer.Run(done)
	resp, err := http.Get("http://data.githubarchive.org/2012-12-10-15.json.gz")
	if err != nil || resp == nil {
		panic("Could not download data")
	}
	defer resp.Body.Close()
	if err != nil {
		log.Println(err)
		return
	}
	gzReader, err := gzip.NewReader(resp.Body)
	defer gzReader.Close()
	if err != nil {
		panic(err)
	}
	r := bufio.NewReader(gzReader)
	var ge GithubEvent
	docsm := make(map[string]bool)
	h := md5.New()
	for {
		line, err := r.ReadBytes('\n')
		if err != nil && err != io.EOF {
			log.Println("FATAL:  could not read line? ", err)
		} else if err != nil {
			indexer.Flush()
			break
		}
		if err := json.Unmarshal(line, &ge); err == nil {
			// create an "ID"
			h.Write(line)
			id := fmt.Sprintf("%x", h.Sum(nil))
			if _, ok := docsm[id]; ok {
				log.Println("HM, already exists? ", ge.Url)
			}
			docsm[id] = true
			indexer.Index("github", ge.Type, id, "", &ge.Created, line, true)
			docCt++
		} else {
			log.Println("ERROR? ", string(line))
		}
	}
	if errCt != 0 {
		log.Println("FATAL, could not load ", errCt)
	}
	// lets wait a bit to ensure that elasticsearch finishes?
	time.Sleep(time.Second * 5)
	if len(docsm) != docCt {
		panic(fmt.Sprintf("Docs didn't match?   %d:%d", len(docsm), docCt))
	}
}
