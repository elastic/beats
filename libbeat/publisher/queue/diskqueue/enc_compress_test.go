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

package diskqueue

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncryptionCompressionRoundTrip(t *testing.T) {
	tests := map[string]struct {
		plaintext []byte
	}{
		"1 rune":     {plaintext: []byte("a")},
		"16 runes":   {plaintext: []byte("bbbbbbbbbbbbbbbb")},
		"17 runes":   {plaintext: []byte("ccccccccccccccccc")},
		"small json": {plaintext: []byte("{\"message\":\"2 123456789010 eni-1235b8ca123456789 - - - - - - - 1431280876 1431280934 - NODATA\"}")},
		"large json": {plaintext: []byte("{\"message\":\"{\\\"CacheCacheStatus\\\":\\\"hit\\\",\\\"CacheResponseBytes\\\":26888,\\\"CacheResponseStatus\\\":200,\\\"CacheTieredFill\\\":true,\\\"ClientASN\\\":1136,\\\"ClientCountry\\\":\\\"nl\\\",\\\"ClientDeviceType\\\":\\\"desktop\\\",\\\"ClientIP\\\":\\\"89.160.20.156\\\",\\\"ClientIPClass\\\":\\\"noRecord\\\",\\\"ClientRequestBytes\\\":5324,\\\"ClientRequestHost\\\":\\\"eqlplayground.io\\\",\\\"ClientRequestMethod\\\":\\\"GET\\\",\\\"ClientRequestPath\\\":\\\"/40865/bundles/plugin/securitySolution/8.0.0/securitySolution.chunk.9.js\\\",\\\"ClientRequestProtocol\\\":\\\"HTTP/1.1\\\",\\\"ClientRequestReferer\\\":\\\"https://eqlplayground.io/s/eqldemo/app/security/timelines/default?sourcerer=(default:!(.siem-signals-eqldemo))&timerange=(global:(linkTo:!(),timerange:(from:%272021-03-03T19:55:15.519Z%27,fromStr:now-24h,kind:relative,to:%272021-03-04T19:55:15.519Z%27,toStr:now)),timeline:(linkTo:!(),timerange:(from:%272020-03-04T19:55:28.684Z%27,fromStr:now-1y,kind:relative,to:%272021-03-04T19:55:28.692Z%27,toStr:now)))&timeline=(activeTab:eql,graphEventId:%27%27,id:%2769f93840-7d23-11eb-866c-79a0609409ba%27,isOpen:!t)\\\",\\\"ClientRequestURI\\\":\\\"/40865/bundles/plugin/securitySolution/8.0.0/securitySolution.chunk.9.js\\\",\\\"ClientRequestUserAgent\\\":\\\"Mozilla/5.0(WindowsNT10.0;Win64;x64)AppleWebKit/537.36(KHTML,likeGecko)Chrome/91.0.4472.124Safari/537.36\\\",\\\"ClientSSLCipher\\\":\\\"NONE\\\",\\\"ClientSSLProtocol\\\":\\\"none\\\",\\\"ClientSrcPort\\\":0,\\\"ClientXRequestedWith\\\":\\\"\\\",\\\"EdgeColoCode\\\":\\\"33.147.138.217\\\",\\\"EdgeColoID\\\":20,\\\"EdgeEndTimestamp\\\":1625752958875000000,\\\"EdgePathingOp\\\":\\\"wl\\\",\\\"EdgePathingSrc\\\":\\\"macro\\\",\\\"EdgePathingStatus\\\":\\\"nr\\\",\\\"EdgeRateLimitAction\\\":\\\"\\\",\\\"EdgeRateLimitID\\\":0,\\\"EdgeRequestHost\\\":\\\"eqlplayground.io\\\",\\\"EdgeResponseBytes\\\":24743,\\\"EdgeResponseCompressionRatio\\\":0,\\\"EdgeResponseContentType\\\":\\\"application/javascript\\\",\\\"EdgeResponseStatus\\\":200,\\\"EdgeServerIP\\\":\\\"89.160.20.156\\\",\\\"EdgeStartTimestamp\\\":1625752958812000000,\\\"FirewallMatchesActions\\\":[],\\\"FirewallMatchesRuleIDs\\\":[],\\\"FirewallMatchesSources\\\":[],\\\"OriginIP\\\":\\\"\\\",\\\"OriginResponseBytes\\\":0,\\\"OriginResponseHTTPExpires\\\":\\\"\\\",\\\"OriginResponseHTTPLastModified\\\":\\\"\\\",\\\"OriginResponseStatus\\\":0,\\\"OriginResponseTime\\\":0,\\\"OriginSSLProtocol\\\":\\\"unknown\\\",\\\"ParentRayID\\\":\\\"66b9d9f88b5b4c4f\\\",\\\"RayID\\\":\\\"66b9d9f890ae4c4f\\\",\\\"SecurityLevel\\\":\\\"off\\\",\\\"WAFAction\\\":\\\"unknown\\\",\\\"WAFFlags\\\":\\\"0\\\",\\\"WAFMatchedVar\\\":\\\"\\\",\\\"WAFProfile\\\":\\\"unknown\\\",\\\"WAFRuleID\\\":\\\"\\\",\\\"WAFRuleMessage\\\":\\\"\\\",\\\"WorkerCPUTime\\\":0,\\\"WorkerStatus\\\":\\\"unknown\\\",\\\"WorkerSubrequest\\\":true,\\\"WorkerSubrequestCount\\\":0,\\\"ZoneID\\\":393347122}\"}")},
	}

	for name, tc := range tests {
		pr, pw := io.Pipe()
		key := []byte("keykeykeykeykeyk")
		src := bytes.NewReader(tc.plaintext)
		var dst bytes.Buffer
		var tEncBuf bytes.Buffer
		var tCompBuf bytes.Buffer

		go func() {
			ew, err := NewEncryptionWriter(NopWriteCloseSyncer(pw), key)
			assert.Nil(t, err, name)
			cw := NewCompressionWriter(ew)
			_, err = io.Copy(cw, src)
			assert.Nil(t, err, name)
			err = cw.Close()
			assert.Nil(t, err, name)
		}()

		ter := io.TeeReader(pr, &tEncBuf)
		er, err := NewEncryptionReader(io.NopCloser(ter), key)
		assert.Nil(t, err, name)

		tcr := io.TeeReader(er, &tCompBuf)

		cr := NewCompressionReader(io.NopCloser(tcr))

		_, err = io.Copy(&dst, cr)
		assert.Nil(t, err, name)
		// Check round trip worked
		assert.Equal(t, tc.plaintext, dst.Bytes(), name)
		// Check that cipher text and plaintext don't match
		assert.NotEqual(t, tc.plaintext, tEncBuf.Bytes(), name)
		// Check that compressed text and plaintext don't match
		assert.NotEqual(t, tc.plaintext, tCompBuf.Bytes(), name)
		// Check that compressed text and ciphertext don't match
		assert.NotEqual(t, tEncBuf.Bytes(), tCompBuf.Bytes(), name)
	}
}
