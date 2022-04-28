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

package add_cloud_metadata

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/libbeat/common"
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

const (
	ec2InstanceIdentityURI            = "/2014-02-25/dynamic/instance-identity/document"
	ec2InstanceIMDSv2TokenValueHeader = "X-aws-ec2-metadata-token"
	ec2InstanceIMDSv2TokenTTLHeader   = "X-aws-ec2-metadata-token-ttl-seconds"
	ec2InstanceIMDSv2TokenTTLValue    = "21600"
	ec2InstanceIMDSv2TokenURI         = "/latest/api/token"
)

// fetches IMDSv2 token, returns empty one on errors
func getIMDSv2Token(c *common.Config) string {
	logger := logp.NewLogger("add_cloud_metadata")

	config := defaultConfig()
	if err := c.Unpack(&config); err != nil {
		logger.Warnf("error when load config for getting IMDSv2 token: %s. No token in the metadata request will be used.", err)
		return ""
	}

	tlsConfig, err := tlscommon.LoadTLSConfig(config.TLS)
	if err != nil {
		logger.Warnf("error when load TLS config for getting IMDSv2 token: %s. No token in the metadata request will be used.", err)
		return ""
	}

	client := http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			DialContext: (&net.Dialer{
				Timeout:   config.Timeout,
				KeepAlive: 0,
			}).DialContext,
			TLSClientConfig: tlsConfig.ToConfig(),
		},
	}

	tokenReq, err := http.NewRequest("PUT", fmt.Sprintf("http://%s%s", metadataHost, ec2InstanceIMDSv2TokenURI), nil)
	if err != nil {
		logger.Warnf("error when make token request for getting IMDSv2 token: %s. No token in the metadata request will be used.", err)
		return ""
	}

	tokenReq.Header.Add(ec2InstanceIMDSv2TokenTTLHeader, ec2InstanceIMDSv2TokenTTLValue)
	rsp, err := client.Do(tokenReq)
	if rsp == nil {
		logger.Warnf("read token request for getting IMDSv2 token returns empty: %s. No token in the metadata request will be used.", err)
		return ""
	}

	defer func(body io.ReadCloser) {
		if body != nil {
			body.Close()
		}
	}(rsp.Body)

	if err != nil {
		logger.Warnf("error when read token request for getting IMDSv2 token: %s. No token in the metadata request will be used.", err)
		return ""
	}

	if rsp.StatusCode != http.StatusOK {
		logger.Warnf("error when check request status for getting IMDSv2 token: http request status %d. No token in the metadata request will be used.", rsp.StatusCode)
		return ""
	}

	all, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		logger.Warnf("error when reading token request for getting IMDSv2 token: %s. No token in the metadata request will be used.", err)
		return ""
	}

	return string(all)
}

// AWS EC2 Metadata Service
var ec2MetadataFetcher = provider{
	Name: "aws-ec2",

	Local: true,

	Create: func(_ string, config *common.Config) (metadataFetcher, error) {
		ec2Schema := func(m map[string]interface{}) mapstr.M {
			m["serviceName"] = "EC2"
			out, _ := s.Schema{
				"instance":          s.Object{"id": c.Str("instanceId")},
				"machine":           s.Object{"type": c.Str("instanceType")},
				"region":            c.Str("region"),
				"availability_zone": c.Str("availabilityZone"),
				"service": s.Object{
					"name": c.Str("serviceName"),
				},
				"account": s.Object{"id": c.Str("accountId")},
				"image":   s.Object{"id": c.Str("imageId")},
			}.Apply(m)
			return mapstr.M{"cloud": out}
		}

		headers := make(map[string]string, 1)
		token := getIMDSv2Token(config)
		if len(token) > 0 {
			headers[ec2InstanceIMDSv2TokenValueHeader] = token
		}

		fetcher, err := newMetadataFetcher(config, "aws", headers, metadataHost, ec2Schema, ec2InstanceIdentityURI)
		return fetcher, err
	},
}
