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

func initQCloudTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/meta-data/instance-id" {
			w.Write([]byte("ins-qcloudv5"))
			return
		}
		if r.RequestURI == "/meta-data/placement/region" {
			w.Write([]byte("china-south-gz"))
			return
		}
		if r.RequestURI == "/meta-data/placement/zone" {
			w.Write([]byte("gz-azone2"))
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
}

func TestRetrieveQCloudMetadata(t *testing.T) {
	logp.TestingSetup()

	server := initQCloudTestServer()
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
				"provider":          "qcloud",
				"instance_id":       "ins-qcloudv5",
				"region":            "china-south-gz",
				"availability_zone": "gz-azone2",
			},
		},
	}
	assert.Equal(t, expected, actual.Fields)
}
