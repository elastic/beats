package elasticsearch

import (
	"net/http"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/client/metadata"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
	"io/ioutil"
	"bytes"
	"github.com/elastic/beats/libbeat/logp"
	"time"
	"io"
	"strings"
)

type AwsData struct {
	Config     aws.Config
	ClientInfo metadata.ClientInfo
}

// Calculate signature for request and append headers
func (conn *Connection) performAwsSignature(req *http.Request) {

	if conn.Aws == nil {
		return
	}

	signer := v4.Signer{
		Credentials: conn.Aws.Config.Credentials,
		Logger:      aws.NewDefaultLogger(), // TODO remove debug logging
		Debug:       aws.LogDebugWithSigning,
	}

	var reader io.ReadSeeker
	if req.Body != nil {
		body := req.Body
		buf, err := ioutil.ReadAll(body)
		if err != nil {
			logp.Err("Error performing AWS signature: %s", err)
			return
		}
		reader = bytes.NewReader(buf)
		req.Body = ioutil.NopCloser(reader)
	} else {
		reader = nil
	}

	// AWS-SDK creates invalid signature when host contains a port
	// See https://github.com/aws/aws-sdk-go/issues/1537
	// Host manipulations can be removed when it's fixed
	host := req.Host
	colon := strings.IndexByte(req.Host, ':')
	if colon >= 0 {
		req.Host = req.Host[:colon]
	}
	_, err := signer.Sign(req, reader, "es", *conn.Aws.Config.Region, time.Now())
	if err != nil {
		logp.Err("Error performing AWS signature: %s", err)
	}
	req.Host = host
}

func (c *elasticsearchConfig) Aws() *AwsData {
	if c.AwsConfig.Enabled {
		credentials := defaults.CredChain(defaults.Config(), defaults.Handlers())
		if _, err := credentials.Get(); err != nil {
			logp.Err("Error loading AWS credentials: %s", err)
			return nil
		}

		region := c.AwsConfig.Region
		if region == "" {
			region = "us-east-1"
		}

		return &AwsData{
			Config: *aws.NewConfig().WithCredentials(credentials).WithRegion(region),
		}
	}

	return nil
}
