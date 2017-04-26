package datastore

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/vic/pkg/vsphere/simulator"
	"github.com/vmware/vic/pkg/vsphere/simulator/esx"
)

func TestFetchEventContents(t *testing.T) {

	s := simulator.New(simulator.NewServiceInstance(esx.ServiceContent, esx.RootFolder))

	ts := s.NewServer()
	defer ts.Close()

	// First create a local datastore to test metric
	tmpDir :=createDatastore(ts, t)

	urlSimulator := ts.URL.Scheme + "://" + ts.URL.Host + ts.URL.Path

	config := map[string]interface{}{
		"module":     "vsphere",
		"metricsets": []string{"datastore"},
		"hosts":      []string{urlSimulator},
		"username":   "user",
		"password":   "pass",
		"insecure":   true,
	}

	f := mbtest.NewEventsFetcher(t, config)

	events, err := f.Fetch()

	event := events[0]
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	assert.EqualValues(t, "ha-datacenter", event["datacenter"])
	assert.EqualValues(t, "test", event["name"])
	assert.EqualValues(t, "local", event["fstype"])

	capacity := event["capacity"].(common.MapStr)

	capacityTotal := capacity["total"].(common.MapStr)
	assert.True(t, (capacityTotal["bytes"].(int64) > 1410745958))

	capacityFree := capacity["free"].(common.MapStr)
	assert.True(t, (capacityFree["bytes"].(int64) > 110715289))

	capacityUsed := capacity["used"].(common.MapStr)
	assert.True(t, (capacityUsed["bytes"].(int64) > 1300030668))
	assert.EqualValues(t, 92, capacityUsed["pct"])

	os.RemoveAll(tmpDir)
}

func createDatastore(ts *simulator.Server, t *testing.T)  string {
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, err := govmomi.NewClient(ctx, ts.URL, true)

        if !assert.NoError(t, err) {
                t.FailNow()
        }

        f := find.NewFinder(c.Client, true)

        // Get all datacenters
        dcs, err := f.DatacenterList(ctx, "*")
        if !assert.NoError(t, err) {
                t.FailNow()
        }

        var tempDir = func() (string, error) {
                return ioutil.TempDir("", "govcsim-")
        }

	dir := ""

	for _, dc := range dcs {
                f.SetDatacenter(dc)

                hss, err := f.HostSystemList(ctx, "*")
        	if !assert.NoError(t, err) {
                	t.FailNow()
        	}	

                dir, err = tempDir()
                if !assert.NoError(t, err) {
                        t.FailNow()
                }

                for _, hs := range hss {
                        dss, err := hs.ConfigManager().DatastoreSystem(ctx)
                        if !assert.NoError(t, err) {
        	                t.FailNow()
	                }
        
                        _, err = dss.CreateLocalDatastore(ctx, "test", dir)

                	if !assert.NoError(t, err) {
                        	t.FailNow()
                	}
                }
        }
	
	return dir
}
