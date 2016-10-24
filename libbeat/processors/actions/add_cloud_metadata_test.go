package actions

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/stretchr/testify/assert"
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

const gceMetadataV1 = `{
  "instance": {
    "attributes": {},
    "cpuPlatform": "Intel Haswell",
    "description": "",
    "disks": [
      {
        "deviceName": "test-gce-dev",
        "index": 0,
        "mode": "READ_WRITE",
        "type": "PERSISTENT"
      }
    ],
    "hostname": "test-gce-dev.c.test-dev.internal",
    "id": 3910564293633576924,
    "image": "",
    "licenses": [
      {
        "id": "1000000"
      }
    ],
    "machineType": "projects/111111111111/machineTypes/f1-micro",
    "maintenanceEvent": "NONE",
    "networkInterfaces": [
      {
        "accessConfigs": [
          {
            "externalIp": "10.10.10.10",
            "type": "ONE_TO_ONE_NAT"
          }
        ],
        "forwardedIps": [],
        "ip": "10.10.0.2",
        "ipAliases": [],
        "mac": "44:00:00:00:00:01",
        "network": "projects/111111111111/networks/default"
      }
    ],
    "scheduling": {
      "automaticRestart": "TRUE",
      "onHostMaintenance": "MIGRATE",
      "preemptible": "FALSE"
    },
    "serviceAccounts": {
      "111111111111-compute@developer.gserviceaccount.com": {
        "aliases": [
          "default"
        ],
        "email": "111111111111-compute@developer.gserviceaccount.com",
        "scopes": [
          "https://www.googleapis.com/auth/devstorage.read_only",
          "https://www.googleapis.com/auth/logging.write",
          "https://www.googleapis.com/auth/monitoring.write",
          "https://www.googleapis.com/auth/servicecontrol",
          "https://www.googleapis.com/auth/service.management.readonly"
        ]
      },
      "default": {
        "aliases": [
          "default"
        ],
        "email": "111111111111-compute@developer.gserviceaccount.com",
        "scopes": [
          "https://www.googleapis.com/auth/devstorage.read_only",
          "https://www.googleapis.com/auth/logging.write",
          "https://www.googleapis.com/auth/monitoring.write",
          "https://www.googleapis.com/auth/servicecontrol",
          "https://www.googleapis.com/auth/service.management.readonly"
        ]
      }
    },
    "tags": [],
    "virtualClock": {
      "driftToken": "0"
    },
    "zone": "projects/111111111111/zones/us-east1-b"
  },
  "project": {
    "attributes": {
      "sshKeys": "developer:ssh-rsa 222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222 google-ssh {\"userName\":\"foo@bar.com\",\"expireOn\":\"2016-10-06T20:20:41+0000\"}\ndev:ecdsa-sha2-nistp256 4444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444= google-ssh {\"userName\":\"foo@bar.com\",\"expireOn\":\"2016-10-06T20:20:40+0000\"}\ndev:ssh-rsa 444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444444 dev"
    },
    "numericProjectId": 111111111111,
    "projectId": "test-dev"
  }
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

func initDigitalOceanTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/metadata/v1.json" {
			w.Write([]byte(digitalOceanMetadataV1))
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
}

func initGCETestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/computeMetadata/v1/?recursive=true&alt=json" {
			w.Write([]byte(gceMetadataV1))
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
}

func TestRetrieveAWSMetadata(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	server := initEC2TestServer()
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
				"provider":          "ec2",
				"instance_id":       "i-11111111",
				"machine_type":      "t2.medium",
				"region":            "us-east-1",
				"availability_zone": "us-east-1c",
			},
		},
	}
	assert.Equal(t, expected, actual)
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

func TestRetrieveGCEMetadata(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	server := initGCETestServer()
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
				"provider":          "gce",
				"instance_id":       "3910564293633576924",
				"machine_type":      "projects/111111111111/machineTypes/f1-micro",
				"availability_zone": "projects/111111111111/zones/us-east1-b",
				"project_id":        "test-dev",
			},
		},
	}
	assert.Equal(t, expected, actual)
}
