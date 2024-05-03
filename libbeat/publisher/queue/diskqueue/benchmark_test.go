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

// Usage:
//
// go test -bench=100k -benchtime 1x -count 10 -timeout 10m -benchmem | tee results.txt && benchstat results.txt
//
// then
//
// benchstat results.txt
//
// you can give benchstat multiple files to analyse and it will
// compare the results between them.
// https://pkg.go.dev/golang.org/x/perf/cmd/benchstat

package diskqueue

import (
	"math/rand"
	"testing"
	"time"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-shipper-client/pkg/proto/messages"
)

var (
	// constant event time
	eventTime = time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	//sample event messages, so size of every frame isn't identical
	msgs = []string{
		"192.168.33.1 - - [26/Dec/2016:16:22:00 +0000] \"GET / HTTP/1.1\" 200 484 \"-\" \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/54.0.2840.98 Safari/537.36\"",
		"{\"eventVersion\":\"1.05\",\"userIdentity\":{\"type\":\"IAMUser\",\"principalId\":\"EXAMPLE_ID\",\"arn\":\"arn:aws:iam::0123456789012:user/Alice\",\"accountId\":\"0123456789012\",\"accessKeyId\":\"EXAMPLE_KEY\",\"userName\":\"Alice\",\"sessionContext\":{\"sessionIssuer\":{},\"webIdFederationData\":{},\"attributes\":{\"mfaAuthenticated\":\"true\",\"creationDate\":\"2020-01-08T15:12:16Z\"}},\"invokedBy\":\"signin.amazonaws.com\"},\"eventTime\":\"2020-01-08T20:58:45Z\",\"eventSource\":\"cloudtrail.amazonaws.com\",\"eventName\":\"UpdateTrail\",\"awsRegion\":\"us-west-2\",\"sourceIPAddress\":\"127.0.0.1\",\"userAgent\":\"signin.amazonaws.com\",\"requestParameters\":{\"name\":\"arn:aws:cloudtrail:us-west-2:0123456789012:trail/TEST-trail\",\"s3BucketName\":\"test-cloudtrail-bucket\",\"snsTopicName\":\"\",\"isMultiRegionTrail\":true,\"enableLogFileValidation\":false,\"kmsKeyId\":\"\"},\"responseElements\":{\"name\":\"TEST-trail\",\"s3BucketName\":\"test-cloudtrail-bucket\",\"snsTopicName\":\"\",\"snsTopicARN\":\"\",\"includeGlobalServiceEvents\":true,\"isMultiRegionTrail\":true,\"trailARN\":\"arn:aws:cloudtrail:us-west-2:0123456789012:trail/TEST-trail\",\"logFileValidationEnabled\":false,\"isOrganizationTrail\":false},\"requestID\":\"EXAMPLE-f3da-42d1-84f5-EXAMPLE\",\"eventID\":\"EXAMPLE-b5e9-4846-8407-EXAMPLE\",\"readOnly\":false,\"eventType\":\"AwsApiCall\",\"recipientAccountId\":\"0123456789012\"}",
		"{\"CacheCacheStatus\":\"hit\",\"CacheResponseBytes\":26888,\"CacheResponseStatus\":200,\"CacheTieredFill\":true,\"ClientASN\":1136,\"ClientCountry\":\"nl\",\"ClientDeviceType\":\"desktop\",\"ClientIP\":\"89.160.20.156\",\"ClientIPClass\":\"noRecord\",\"ClientRequestBytes\":5324,\"ClientRequestHost\":\"eqlplayground.io\",\"ClientRequestMethod\":\"GET\",\"ClientRequestPath\":\"/40865/bundles/plugin/securitySolution/8.0.0/securitySolution.chunk.9.js\",\"ClientRequestProtocol\":\"HTTP/1.1\",\"ClientRequestReferer\":\"https://eqlplayground.io/s/eqldemo/app/security/timelines/default?sourcerer=(default:!(.siem-signals-eqldemo))&timerange=(global:(linkTo:!(),timerange:(from:%272021-03-03T19:55:15.519Z%27,fromStr:now-24h,kind:relative,to:%272021-03-04T19:55:15.519Z%27,toStr:now)),timeline:(linkTo:!(),timerange:(from:%272020-03-04T19:55:28.684Z%27,fromStr:now-1y,kind:relative,to:%272021-03-04T19:55:28.692Z%27,toStr:now)))&timeline=(activeTab:eql,graphEventId:%27%27,id:%2769f93840-7d23-11eb-866c-79a0609409ba%27,isOpen:!t)\",\"ClientRequestURI\":\"/40865/bundles/plugin/securitySolution/8.0.0/securitySolution.chunk.9.js\",\"ClientRequestUserAgent\":\"Mozilla/5.0(WindowsNT10.0;Win64;x64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/91.0.4472.124Safari/537.36\",\"ClientSSLCipher\":\"NONE\",\"ClientSSLProtocol\":\"none\",\"ClientSrcPort\":0,\"ClientXRequestedWith\":\"\",\"EdgeColoCode\":\"33.147.138.217\",\"EdgeColoID\":20,\"EdgeEndTimestamp\":1625752958875000000,\"EdgePathingOp\":\"wl\",\"EdgePathingSrc\":\"macro\",\"EdgePathingStatus\":\"nr\",\"EdgeRateLimitAction\":\"\",\"EdgeRateLimitID\":0,\"EdgeRequestHost\":\"eqlplayground.io\",\"EdgeResponseBytes\":24743,\"EdgeResponseCompressionRatio\":0,\"EdgeResponseContentType\":\"application/javascript\",\"EdgeResponseStatus\":200,\"EdgeServerIP\":\"89.160.20.156\",\"EdgeStartTimestamp\":1625752958812000000,\"FirewallMatchesActions\":[],\"FirewallMatchesRuleIDs\":[],\"FirewallMatchesSources\":[],\"OriginIP\":\"\",\"OriginResponseBytes\":0,\"OriginResponseHTTPExpires\":\"\",\"OriginResponseHTTPLastModified\":\"\",\"OriginResponseStatus\":0,\"OriginResponseTime\":0,\"OriginSSLProtocol\":\"unknown\",\"ParentRayID\":\"66b9d9f88b5b4c4f\",\"RayID\":\"66b9d9f890ae4c4f\",\"SecurityLevel\":\"off\",\"WAFAction\":\"unknown\",\"WAFFlags\":\"0\",\"WAFMatchedVar\":\"\",\"WAFProfile\":\"unknown\",\"WAFRuleID\":\"\",\"WAFRuleMessage\":\"\",\"WorkerCPUTime\":0,\"WorkerStatus\":\"unknown\",\"WorkerSubrequest\":true,\"WorkerSubrequestCount\":0,\"ZoneID\":393347122}",
		"2 123456789010 eni-1235b8ca123456789 - - - - - - - 1431280876 1431280934 - NODATA",
		"Oct 10 2018 12:34:56 localhost CiscoASA[999]: %ASA-6-305012: Teardown dynamic TCP translation from inside:172.31.98.44/1459 to outside:192.168.98.44/8272 duration 0:00:30",
		"{\"timestamp\":\"2021-01-27T01:28:11.488362+0100\",\"flow_id\":1805461738637437,\"in_iface\":\"enp6s0\",\"event_type\":\"alert\",\"src_ip\":\"175.16.199.1\",\"src_port\":80,\"dest_ip\":\"10.31.64.240\",\"dest_port\":47592,\"proto\":\"TCP\",\"ether\":{\"src_mac\":\"00:03:2d:3f:e5:63\",\"dest_mac\":\"00:1b:17:00:01:18\"},\"alert\":{\"action\":\"allowed\",\"gid\":1,\"signature_id\":2100498,\"rev\":7,\"signature\":\"GPL ATTACK_RESPONSE id check returned root\",\"category\":\"Potentially Bad Traffic\",\"severity\":2,\"metadata\":{\"created_at\":[\"2010_09_23\"],\"updated_at\":[\"2010_09_23\"]}},\"http\":{\"hostname\":\"testmynids.org\",\"url\":\"/uid/index.html\",\"http_user_agent\":\"curl/7.58.0\",\"http_content_type\":\"text/html\",\"http_method\":\"GET\",\"protocol\":\"HTTP/1.1\",\"status\":200,\"length\":39},\"app_proto\":\"http\",\"flow\":{\"pkts_toserver\":6,\"pkts_toclient\":5,\"bytes_toserver\":496,\"bytes_toclient\":876,\"start\":\"2021-01-22T23:28:38.673917+0100\"}}",
		"2018-08-28 18:24:25 [10.100.220.70](http://10.100.220.70) GET / - 80 - [10.100.118.31](http://10.100.118.31) Mozilla/4.0+(compatible;+MSIE+7.0;+Windows+NT+6.3;+WOW64;+Trident/7.0;+.NET4.0E;+.NET4.0C;+.NET+CLR+3.5.30729;+.NET+CLR[+2.0.50727](tel:+2050727);+.NET+CLR+3.0.30729) 404 4 2 792",
	}
)

// makePublisherEvent creates a sample publisher.Event, using a random message from msgs list
func makePublisherEvent() publisher.Event {
	return publisher.Event{
		Content: beat.Event{
			Timestamp: eventTime,
			Fields: mapstr.M{
				"message": msgs[rand.Intn(len(msgs))],
			},
		},
	}
}

// makeMessagesEvent creates a sample *messages.Event, using a random message from msgs list
func makeMessagesEvent() *messages.Event {
	return &messages.Event{
		Timestamp: timestamppb.New(eventTime),
		Fields: &messages.Struct{
			Data: map[string]*messages.Value{
				"message": {
					Kind: &messages.Value_StringValue{
						StringValue: msgs[rand.Intn(len(msgs))],
					},
				},
			},
		},
	}
}

// setup creates the disk queue, including a temporary directory to
// hold the queue.  Location of the temporary directory is stored in
// the queue settings.  Call `cleanup` when done with the queue to
// close the queue and remove the temp dir.
func setup(b *testing.B, encrypt bool, compress bool, protobuf bool) (*diskQueue, queue.Producer) {
	s := DefaultSettings()
	s.Path = b.TempDir()
	if encrypt {
		s.EncryptionKey = []byte("testtesttesttest")
	}
	s.UseCompression = compress
	s.UseProtobuf = protobuf
	q, err := NewQueue(logp.L(), nil, s, nil)
	if err != nil {
		panic(err)
	}
	p := q.Producer(queue.ProducerConfig{})

	b.Cleanup(func() {
		err := q.Close()
		if err != nil {
			panic(err)
		}
	})

	return q, p
}

func publishEvents(p queue.Producer, num int, protobuf bool) {
	for i := 0; i < num; i++ {
		var e queue.Entry
		if protobuf {
			e = makeMessagesEvent()
		} else {
			e = makePublisherEvent()
		}
		_, ok := p.Publish(e)
		if !ok {
			panic("didn't publish")
		}
	}
}

func getAndAckEvents(q *diskQueue, num_events int, batch_size int) error {
	var received int
	for {
		batch, err := q.Get(batch_size, 0)
		if err != nil {
			return err
		}
		batch.Done()
		received = received + batch.Count()
		if received == num_events {
			return nil
		}
	}
}

// produceAndConsume generates and publishes events in a go routine, in
// the main go routine it consumes and acks them.  This interleaves
// publish and consume.
func produceAndConsume(p queue.Producer, q *diskQueue, num_events int, batch_size int, protobuf bool) error {
	go publishEvents(p, num_events, protobuf)
	return getAndAckEvents(q, num_events, batch_size)
}

// produceThenConsume generates and publishes events, when all events
// are published it consumes and acks them.
func produceThenConsume(p queue.Producer, q *diskQueue, num_events int, batch_size int, protobuf bool) error {
	publishEvents(p, num_events, protobuf)
	return getAndAckEvents(q, num_events, batch_size)
}

// benchmarkQueue is a wrapper for produceAndConsume, it tries to limit
// timers to just produceAndConsume
func benchmarkQueue(num_events int, batch_size int, encrypt bool, compress bool, async bool, protobuf bool, b *testing.B) {
	b.ResetTimer()
	var err error

	for n := 0; n < b.N; n++ {
		b.StopTimer()
		rand.Seed(1)
		q, p := setup(b, encrypt, compress, protobuf)
		b.StartTimer()
		if async {
			if err = produceAndConsume(p, q, num_events, batch_size, protobuf); err != nil {
				break
			}
		} else {
			if err = produceThenConsume(p, q, num_events, batch_size, protobuf); err != nil {
				break
			}
		}
	}
	if err != nil {
		b.Errorf("Error producing/consuming events: %v", err)
	}
}

// Async benchmarks
func BenchmarkAsync1k(b *testing.B) {
	benchmarkQueue(1000, 10, false, false, true, false, b)
}
func BenchmarkAsync100k(b *testing.B) {
	benchmarkQueue(100000, 1000, false, false, true, false, b)
}
func BenchmarkEncryptAsync1k(b *testing.B) {
	benchmarkQueue(1000, 10, true, false, true, false, b)
}
func BenchmarkEncryptAsync100k(b *testing.B) {
	benchmarkQueue(100000, 1000, true, false, true, false, b)
}
func BenchmarkCompressAsync1k(b *testing.B) {
	benchmarkQueue(1000, 10, false, true, true, false, b)
}
func BenchmarkCompressAsync100k(b *testing.B) {
	benchmarkQueue(100000, 1000, false, true, true, false, b)
}
func BenchmarkEncryptCompressAsync1k(b *testing.B) {
	benchmarkQueue(1000, 10, true, true, true, false, b)
}
func BenchmarkEncryptCompressAsync100k(b *testing.B) {
	benchmarkQueue(100000, 1000, true, true, true, false, b)
}
func BenchmarkProtoAsync1k(b *testing.B) {
	benchmarkQueue(1000, 10, false, false, true, true, b)
}
func BenchmarkProtoAsync100k(b *testing.B) {
	benchmarkQueue(100000, 1000, false, false, true, true, b)
}
func BenchmarkEncCompProtoAsync1k(b *testing.B) {
	benchmarkQueue(1000, 10, true, true, true, true, b)
}
func BenchmarkEncCompProtoAsync100k(b *testing.B) {
	benchmarkQueue(100000, 1000, true, true, true, true, b)
}

// Sync Benchmarks
func BenchmarkSync1k(b *testing.B) {
	benchmarkQueue(1000, 10, false, false, false, false, b)
}
func BenchmarkSync100k(b *testing.B) {
	benchmarkQueue(100000, 1000, false, false, false, false, b)
}
func BenchmarkEncryptSync1k(b *testing.B) {
	benchmarkQueue(1000, 10, true, false, false, false, b)
}
func BenchmarkEncryptSync100k(b *testing.B) {
	benchmarkQueue(100000, 1000, true, false, false, false, b)
}
func BenchmarkCompressSync1k(b *testing.B) {
	benchmarkQueue(1000, 10, false, true, false, false, b)
}
func BenchmarkCompressSync100k(b *testing.B) {
	benchmarkQueue(100000, 1000, false, true, false, false, b)
}
func BenchmarkEncryptCompressSync1k(b *testing.B) {
	benchmarkQueue(1000, 10, true, true, false, false, b)
}
func BenchmarkEncryptCompressSync100k(b *testing.B) {
	benchmarkQueue(100000, 1000, true, true, false, false, b)
}
func BenchmarkProtoSync1k(b *testing.B) {
	benchmarkQueue(1000, 10, false, false, false, true, b)
}
func BenchmarkProtoSync100k(b *testing.B) {
	benchmarkQueue(100000, 1000, false, false, false, true, b)
}
func BenchmarkEncCompProtoSync1k(b *testing.B) {
	benchmarkQueue(1000, 10, true, true, false, true, b)
}
func BenchmarkEncCompProtoSync100k(b *testing.B) {
	benchmarkQueue(100000, 1000, true, true, false, true, b)
}
