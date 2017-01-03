package perf

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/ceph"
	"github.com/elastic/beats/libbeat/logp"
        "fmt"
        "io/ioutil"
        "strings"
        "os/exec"
        "path/filepath"
        "bytes"
	"encoding/json"
)

const (
        measurement = "ceph"
        typeMon     = "monitor"
        typeOsd     = "osd"
        osdPrefix   = "ceph-osd"
        monPrefix   = "ceph-mon"
        sockSuffix  = "asok"
)


// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("ceph", "perf", New); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	cfg *ceph.CephConfig
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	config := ceph.Config{}


	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	logp.Warn("Teste %s",config.CEPH.BinaryPath)

	return &MetricSet{
		BaseMetricSet: base,
		cfg: config.CEPH,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	logp.Warn("Entrando")

	sockets, err := findSockets(m.cfg)
        if err != nil {
                return nil,err
        }

	myEvents := []common.MapStr{}

        for _, s := range sockets {
                dump, err := perfDump(m.cfg.BinaryPath, s)
                if err != nil {
                        continue
                }

		data := make(map[string]interface{})
		errJson := json.Unmarshal([]byte(dump), &data)
		if errJson != nil {
			logp.Warn("Json error")
		}	


		for tag, datapoints := range data {
			event := common.MapStr{
				tag: datapoints,
			}
			myEvents = append(myEvents, event)
		}



		//event := common.MapStr{
		//	"testeamanda": dump,
		//}	
		
        }


	return myEvents, nil
}







var findSockets = func(c *ceph.CephConfig) ([]*socket, error) {
        listing, err := ioutil.ReadDir(c.SocketDir)
        if err != nil {
                return []*socket{}, fmt.Errorf("Failed to read socket directory '%s': %v", c.SocketDir, err)
        }
        sockets := make([]*socket, 0, len(listing))
        for _, info := range listing {
                f := info.Name()
                var sockType string
                var sockPrefix string
                if strings.HasPrefix(f, c.MonPrefix) {
                        sockType = typeMon
                        sockPrefix = monPrefix
                }
                if strings.HasPrefix(f, c.OsdPrefix) {
                        sockType = typeOsd
                        sockPrefix = osdPrefix

                }
                if sockType == typeOsd || sockType == typeMon {
                        path := filepath.Join(c.SocketDir, f)
                        sockets = append(sockets, &socket{parseSockId(f, sockPrefix, c.SocketSuffix), sockType, path})
                }
        }
        return sockets, nil
}

func parseSockId(fname, prefix, suffix string) string {
        s := fname
        s = strings.TrimPrefix(s, prefix)
        s = strings.TrimSuffix(s, suffix)
        s = strings.Trim(s, ".-_")
        return s
}

type socket struct {
        sockId   string
        sockType string
        socket   string
}

var perfDump = func(binary string, socket *socket) (string, error) {
        cmdArgs := []string{"--admin-daemon", socket.socket}
        if socket.sockType == typeOsd {
                cmdArgs = append(cmdArgs, "perf", "dump")
        } else if socket.sockType == typeMon {
                cmdArgs = append(cmdArgs, "perfcounters_dump")
        } else {
                return "", fmt.Errorf("ignoring unknown socket type: %s", socket.sockType)
        }


        cmd := exec.Command(binary, cmdArgs...)

        var out bytes.Buffer
        cmd.Stdout = &out
        var stderr bytes.Buffer
        cmd.Stderr = &stderr


        err := cmd.Run()
        if err != nil {
                return "", fmt.Errorf("error running ceph dump: %s", stderr.String())
        }
        return out.String(), nil

}

