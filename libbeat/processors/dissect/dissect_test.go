// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package dissect

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoToken(t *testing.T) {
	_, err := New("hello")
	assert.Equal(t, errInvalidTokenizer, err)
}

func TestEmptyString(t *testing.T) {
	d, err := New("%{hello}")
	_, err = d.Dissect("")
	assert.Equal(t, errEmpty, err)
}

// JSON tags are used to create a common test file for the `logstash-filter-dissect` and the
// beat implementation.
type dissectTest struct {
	Name     string `json:"name"`
	Tok      string `json:"tok"`
	Msg      string `json:"msg"`
	Expected Map    `json:"expected"`
	Skip     bool   `json:"skip"`
	Fail     bool   `json:"fail"`
}

var tests []dissectTest

func init() {
	content, err := ioutil.ReadFile("testdata/dissect_tests.json")
	if err != nil {
		fmt.Printf("could not read the content of 'dissect_tests', error: %s", err)
		os.Exit(1)
	}

	json.Unmarshal(content, &tests)
}

func TestDissect(t *testing.T) {
	for _, test := range tests {
		if test.Skip {
			continue
		}
		t.Run(test.Name, func(t *testing.T) {
			d, err := New(test.Tok)
			if !assert.NoError(t, err) {
				return
			}

			if test.Fail {
				_, err := d.Dissect(test.Msg)
				assert.Error(t, err)
				return
			}

			r, err := d.Dissect(test.Msg)
			if !assert.NoError(t, err) {
				return
			}

			assert.Equal(t, test.Expected, r)
		})
	}
}

var results Map
var o [][]string

func BenchmarkDissect(b *testing.B) {
	for _, test := range tests {
		if test.Skip {
			continue
		}
		b.Run(test.Name, func(b *testing.B) {
			tok := test.Tok
			msg := test.Msg
			d, err := New(tok)
			if !assert.NoError(b, err) {
				return
			}
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				r, err := d.Dissect(msg)
				if test.Fail {
					assert.Error(b, err)
					return
				}
				assert.NoError(b, err)
				results = r
			}
		})
	}

	// Add a few regular expression matches against the same string the test suite,
	// this give us a baseline to compare to, note that we only test a raw match against the string.
	b.Run("Regular expression", func(b *testing.B) {
		re := regexp.MustCompile("/var/log/([a-z]+)/log/([a-z]+)/apache/([a-b]+)")
		by := "/var/log/docker/more/apache/super"
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			o = re.FindAllStringSubmatch(by, -1)
		}
	})

	b.Run("Larger regular expression", func(b *testing.B) {
		re := regexp.MustCompile("^(\\d{2})-(\\w{3})-(\\d{4})\\s([0-9:.]+)\\s(\\w+)\\s\\[([a-zA-Z0-9-]+)\\]\\s([a-zA-Z0-9.]+)\\s(.+)")

		by := `18-Apr-2018 06:53:20.411 INFO [http-nio-8080-exec-1] org.apache.coyote.http11.Http11Processor.service Error parsing HTTP request header
 Note: further occurrences of HTTP header parsing errors will be logged at DEBUG level.
 java.lang.IllegalArgumentException: Invalid character found in method name. HTTP method names must be tokens
    at org.apache.coyote.http11.Http11InputBuffer.parseRequestLine(Http11InputBuffer.java:426)
    at org.apache.coyote.http11.Http11Processor.service(Http11Processor.java:687)
    at org.apache.coyote.AbstractProcessorLight.process(AbstractProcessorLight.java:66)
    at org.apache.coyote.AbstractProtocol$ConnectionHandler.process(AbstractProtocol.java:790)
    at org.apache.tomcat.util.net.NioEndpoint$SocketProcessor.doRun(NioEndpoint.java:1459)
    at org.apache.tomcat.util.net.SocketProcessorBase.run(SocketProcessorBase.java:49)
    at java.util.concurrent.ThreadPoolExecutor.runWorker(ThreadPoolExecutor.java:1149)
    at java.util.concurrent.ThreadPoolExecutor$Worker.run(ThreadPoolExecutor.java:624)
    at org.apache.tomcat.util.threads.TaskThread$WrappingRunnable.run(TaskThread.java:61)
    at java.lang.Thread.run(Thread.java:748)`
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			o = re.FindAllStringSubmatch(by, -1)
		}
	})

	b.Run("regular expression to match end of line", func(b *testing.B) {
		re := regexp.MustCompile("MACHINE\\[(\\w+)\\]$")

		by := `18-Apr-2018 06:53:20.411 INFO [http-nio-8080-exec-1] org.apache.coyote.http11.Http11Processor.service Error parsing HTTP request header
 Note: further occurrences of HTTP header parsing errors will be logged at DEBUG level.
 java.lang.IllegalArgumentException: Invalid character found in method name. HTTP method names must be tokens
    at org.apache.coyote.http11.Http11InputBuffer.parseRequestLine(Http11InputBuffer.java:426)
    at org.apache.coyote.http11.Http11Processor.service(Http11Processor.java:687)
    at org.apache.coyote.AbstractProcessorLight.process(AbstractProcessorLight.java:66)
    at org.apache.coyote.AbstractProtocol$ConnectionHandler.process(AbstractProtocol.java:790)
    at org.apache.tomcat.util.net.NioEndpoint$SocketProcessor.doRun(NioEndpoint.java:1459)
    at org.apache.tomcat.util.net.SocketProcessorBase.run(SocketProcessorBase.java:49)
    at java.util.concurrent.ThreadPoolExecutor.runWorker(ThreadPoolExecutor.java:1149)
    at java.util.concurrent.ThreadPoolExecutor$Worker.run(ThreadPoolExecutor.java:624)
    at org.apache.tomcat.util.threads.TaskThread$WrappingRunnable.run(TaskThread.java:61)
    at java.lang.Thread.run(Thread.java:748) MACHINE[hello]`
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			o = re.FindAllStringSubmatch(by, -1)
		}
	})
}
