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

const azInstanceIdentityDocument = `{
	"location": "eastus2",
	"name": "test-az-vm",
	"offer": "UbuntuServer",
	"osType": "Linux",
	"platformFaultDomain": "0",
	"platformUpdateDomain": "0",
	"publisher": "Canonical",
	"sku": "14.04.4-LTS",
	"version": "14.04.201605091",
	"vmId": "04ab04c3-63de-4709-a9f9-9ab8c0411d5e",
	"vmSize": "Standard_D3_v2"
}`

func initAzureTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/metadata/instance/compute?api-version=2017-04-02" && r.Header.Get("Metadata") == "true" {
			w.Write([]byte(azInstanceIdentityDocument))
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
}

func TestRetrieveAzureMetadata(t *testing.T) {
	logp.TestingSetup()

	server := initAzureTestServer()
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
				"provider":      "az",
				"instance_id":   "04ab04c3-63de-4709-a9f9-9ab8c0411d5e",
				"instance_name": "test-az-vm",
				"machine_type":  "Standard_D3_v2",
				"region":        "eastus2",
			},
		},
	}
	assert.Equal(t, expected, actual.Fields)
}
