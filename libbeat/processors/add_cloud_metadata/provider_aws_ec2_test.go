package add_cloud_metadata

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

const ec2InstanceIdentityDocument = `{
  "devpayProductCodes" : null,
  "privateIp" : "10.0.0.1",
  "availabilityZone" : "us-east-1c",
  "accountId" : "111111111111111",
  "version" : "2010-08-31",
  "instanceId" : "i-11111111",
  "billingProducts" : null,
  "instanceType" : "t2.medium",
  "imageId" : "ami-6869aa05",
  "pendingTime" : "2016-09-20T15:43:02Z",
  "architecture" : "x86_64",
  "kernelId" : null,
  "ramdiskId" : null,
  "region" : "us-east-1"
}`

func initEC2TestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/2014-02-25/dynamic/instance-identity/document" {
			w.Write([]byte(ec2InstanceIdentityDocument))
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
}

func TestRetrieveAWSMetadata(t *testing.T) {
	logp.TestingSetup()

	server := initEC2TestServer()
	defer server.Close()

	config, err := common.NewConfigFrom(map[string]interface{}{
		"host": server.Listener.Addr().String(),
	})
	if err != nil {
		t.Fatal(err)
	}

	p, err := newCloudMetadata(config)
	if err != nil {
		t.Fatal(err)
	}

	actual, err := p.Run(&beat.Event{Fields: common.MapStr{}})
	if err != nil {
		t.Fatal(err)
	}

	expected := common.MapStr{
		"meta": common.MapStr{
			"cloud": common.MapStr{
				"provider":          "ec2",
				"instance_id":       "i-11111111",
				"machine_type":      "t2.medium",
				"region":            "us-east-1",
				"availability_zone": "us-east-1c",
			},
		},
	}
	assert.Equal(t, expected, actual.Fields)
}
