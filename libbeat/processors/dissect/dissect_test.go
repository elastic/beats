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
	"flag"
	"io/ioutil"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

var export = flag.Bool("test.export-dissect", false, "export dissect tests to JSON.")

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

var tests = []dissectTest{
	{
		Name: "When all the defined fields are captured by we have remaining data",
		Tok:  "level=%{level} ts=%{timestamp} caller=%{caller} msg=\"%{message}\"",
		Msg:  "level=info ts=2018-06-27T17:19:13.036579993Z caller=main.go:222 msg=\"Starting OK\" version=\"(version=2.3.1, branch=HEAD, revision=188ca45bd85ce843071e768d855722a9d9dabe03)\"}",
		Expected: Map{
			"level":     "info",
			"timestamp": "2018-06-27T17:19:13.036579993Z",
			"caller":    "main.go:222",
			"message":   "Starting OK",
		},
	},
	{
		Name: "Complex stack trace",
		Tok:  "%{day}-%{month}-%{year} %{hour} %{severity} [%{thread_id}] %{origin} %{message}",
		Msg: `18-Apr-2018 06:53:20.411 INFO [http-nio-8080-exec-1] org.apache.coyote.http11.Http11Processor.service Error parsing HTTP request header
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
    at java.lang.Thread.run(Thread.java:748)`,
		Expected: Map{
			"day":       "18",
			"month":     "Apr",
			"year":      "2018",
			"hour":      "06:53:20.411",
			"severity":  "INFO",
			"thread_id": "http-nio-8080-exec-1",
			"origin":    "org.apache.coyote.http11.Http11Processor.service",
			"message": `Error parsing HTTP request header
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
    at java.lang.Thread.run(Thread.java:748)`,
		},
	},
	{
		Name: "fails when delimiter is not found at the beginning of the string",
		Tok:  "/var/log/%{key}.log",
		Msg:  "foobar",
		Fail: true,
	},
	{
		Name: "fails when delimiter is not found after the key",
		Tok:  "/var/log/%{key}.log",
		Msg:  "/var/log/foobar",
		Fail: true,
	},
	{
		Name:     "simple dissect",
		Tok:      "%{key}",
		Msg:      "foobar",
		Expected: Map{"key": "foobar"},
	},
	{
		Name:     "dissect two replacement",
		Tok:      "%{key1} %{key2}",
		Msg:      "foo bar",
		Expected: Map{"key1": "foo", "key2": "bar"},
	},
	{
		Name:     "one level dissect not end of string",
		Tok:      "/var/%{key}/log",
		Msg:      "/var/foobar/log",
		Expected: Map{"key": "foobar"},
	},
	{
		Name:     "one level dissect",
		Tok:      "/var/%{key}",
		Msg:      "/var/foobar/log",
		Expected: Map{"key": "foobar/log"},
	},
	{
		Name:     "multiple keys dissect end of string",
		Tok:      "/var/%{key}/log/%{key1}",
		Msg:      "/var/foobar/log/apache",
		Expected: Map{"key": "foobar", "key1": "apache"},
	},
	{
		Name:     "multiple keys not end of string",
		Tok:      "/var/%{key}/log/%{key1}.log",
		Msg:      "/var/foobar/log/apache.log",
		Expected: Map{"key": "foobar", "key1": "apache"},
	},
	{
		Name:     "simple ordered",
		Tok:      "%{+key/3} %{+key/1} %{+key/2}",
		Msg:      "1 2 3",
		Expected: Map{"key": "2 3 1"},
	},
	{
		Name:     "simple append",
		Tok:      "%{key}-%{+key}-%{+key}",
		Msg:      "1-2-3",
		Expected: Map{"key": "1-2-3"},
	},
	{
		Name:     "indirect field",
		Tok:      "%{key} %{&key}",
		Msg:      "hello world",
		Expected: Map{"key": "hello", "hello": "world"},
	},
	{
		Name:     "skip field",
		Tok:      "%{} %{key}",
		Msg:      "hello world",
		Expected: Map{"key": "world"},
	},
	{
		Name:     "named skiped field with indirect",
		Tok:      "%{?key} %{&key}",
		Msg:      "hello world",
		Expected: Map{"hello": "world"},
	},
	{
		Name: "missing fields",
		Tok:  "%{name},%{addr1},%{addr2},%{addr3},%{city},%{zip}",
		Msg:  "Jane Doe,4321 Fifth Avenue,,,New York,87432",
		Expected: Map{
			"name":  "Jane Doe",
			"addr1": "4321 Fifth Avenue",
			"addr2": "",
			"addr3": "",
			"city":  "New York",
			"zip":   "87432",
		},
	},
	{
		Name: "ignore right padding",
		Tok:  "%{id} %{function->} %{server}",
		Msg:  "00000043 ViewReceive     machine-321",
		Expected: Map{
			"id":       "00000043",
			"function": "ViewReceive",
			"server":   "machine-321",
		},
	},
	{
		Name: "padding on the last key need a delimiter",
		Tok:  "%{id} %{function} %{server->} ",
		Msg:  "00000043 ViewReceive machine-321    ",
		Expected: Map{
			"id":       "00000043",
			"function": "ViewReceive",
			"server":   "machine-321",
		},
	},
	{
		Name: "ignore left padding",
		Tok:  "%{id->} %{function} %{server}",
		Msg:  "00000043    ViewReceive machine-321",
		Expected: Map{
			"id":       "00000043",
			"function": "ViewReceive",
			"server":   "machine-321",
		},
	},
	{
		Name: "when the delimiters contains `{` and `}`",
		Tok:  "{%{a}}{%{b}} %{rest}",
		Msg:  "{c}{d} anything",
		Expected: Map{
			"a":    "c",
			"b":    "d",
			"rest": "anything",
		},
	},
}

func TestDissect(t *testing.T) {
	if export != nil && *export {
		dumpJSON()
		return
	}

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

func dumpJSON() {
	b, err := json.MarshalIndent(&tests, "", "\t")
	if err != nil {
		panic("could not marshal json")
	}

	err = ioutil.WriteFile("dissect_tests.json", b, 0666)
	if err != nil {
		panic("could not write to file")
	}
}
