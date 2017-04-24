package add_cloud_metadata

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/stretchr/testify/assert"
)

const digitalOceanMetadataV1 = `{
  "droplet_id":1111111,
  "hostname":"sample-droplet",
  "vendor_data":"#cloud-config\ndisable_root: false\nmanage_etc_hosts: true\n\ncloud_config_modules:\n - ssh\n - set_hostname\n - [ update_etc_hosts, once-per-instance ]\n\ncloud_final_modules:\n - scripts-vendor\n - scripts-per-once\n - scripts-per-boot\n - scripts-per-instance\n - scripts-user\n",
  "public_keys":["ssh-rsa 111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111 sammy@digitalocean.com"],
  "region":"nyc3",
  "interfaces":{
    "private":[
      {
        "ipv4":{
          "ip_address":"10.0.0.2",
          "netmask":"255.255.0.0",
          "gateway":"10.10.0.1"
        },
        "mac":"54:11:00:00:00:00",
        "type":"private"
      }
    ],
    "public":[
      {
        "ipv4":{
          "ip_address":"192.168.20.105",
          "netmask":"255.255.192.0",
          "gateway":"192.168.20.1"
        },
        "ipv6":{
          "ip_address":"1111:1111:0000:0000:0000:0000:0000:0000",
          "cidr":64,
          "gateway":"0000:0000:0800:0010:0000:0000:0000:0001"
        },
        "mac":"34:00:00:ff:00:00",
        "type":"public"}
    ]
  },
  "floating_ip": {
    "ipv4": {
      "active": false
    }
  },
  "dns":{
    "nameservers":[
      "2001:4860:4860::8844",
      "2001:4860:4860::8888",
      "8.8.8.8"
    ]
  }
}`

func initDigitalOceanTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/metadata/v1.json" {
			w.Write([]byte(digitalOceanMetadataV1))
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
}

func TestRetrieveDigitalOceanMetadata(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	server := initDigitalOceanTestServer()
	defer server.Close()

	config, err := common.NewConfigFrom(map[string]interface{}{
		"host": server.Listener.Addr().String(),
	})
	if err != nil {
		t.Fatal(err)
	}

	p, err := newCloudMetadata(*config)
	if err != nil {
		t.Fatal(err)
	}

	actual, err := p.Run(common.MapStr{})
	if err != nil {
		t.Fatal(err)
	}

	expected := common.MapStr{
		"meta": common.MapStr{
			"cloud": common.MapStr{
				"provider":    "digitalocean",
				"instance_id": "1111111",
				"region":      "nyc3",
			},
		},
	}
	assert.Equal(t, expected, actual)
}
