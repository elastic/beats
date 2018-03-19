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

func initECSTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/latest/meta-data/instance-id" {
			w.Write([]byte("i-wz9g2hqiikg0aliyun2b"))
			return
		}
		if r.RequestURI == "/latest/meta-data/region-id" {
			w.Write([]byte("cn-shenzhen"))
			return
		}
		if r.RequestURI == "/latest/meta-data/zone-id" {
			w.Write([]byte("cn-shenzhen-a"))
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
}

func TestRetrieveAlibabaCloudMetadata(t *testing.T) {
	logp.TestingSetup()

	server := initECSTestServer()
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
				"provider":          "ecs",
				"instance_id":       "i-wz9g2hqiikg0aliyun2b",
				"region":            "cn-shenzhen",
				"availability_zone": "cn-shenzhen-a",
			},
		},
	}
	assert.Equal(t, expected, actual.Fields)
}
