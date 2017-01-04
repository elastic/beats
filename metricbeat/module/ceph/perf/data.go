package perf

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
        "fmt"
        "os/exec"
        "bytes"
	"encoding/json"
	"strings"
)

const (
        measurement = "ceph"
        typeMon     = "monitor"
        typeOsd     = "osd"
        osdPrefix   = "ceph-osd"
        monPrefix   = "ceph-mon"
        sockSuffix  = "asok"
)

func eventsMapping(socketsList []*socket,binaryPath string) []common.MapStr {
	myEvents := []common.MapStr{}

	for _, socket := range socketsList {

                dump, err := perfDump(binaryPath, socket)
                if err != nil {
                        logp.Err("An error occurred while reading sockets for getting ceph perf: %v", err)
                        continue
                }

                data, err := parseDump(dump)
                if err != nil {
                        logp.Err("An error occurred while parsing data for getting ceph perf: %v", err)
                        continue
                }

		for tag, datapoints := range data {
			
                        event := common.MapStr{
                                formatTagName(tag): datapoints,
                        }

                        myEvents = append(myEvents, event)
                }
	}

	return myEvents
}

func formatTagName(oldtag string) (string){

	// Replace '::' fields
	r := strings.NewReplacer("::",".",":.",".",":",".")
	logp.Warn("newtag: %s", r.Replace(oldtag))

	return r.Replace(oldtag)

}


/*func formatDataPointName(oldDatapoint string) (string){

        // Remove repeated fields
        // Example:
        // leveldb.leveldb_submit_latency.avgcount must be leveldb.submit_latency.avgcount

        splitedDataPoint := strings.Split(oldDatapoint, ".")

        newDatapoint := splitedDataPoint[0]
        oldslice := ""

        for i, slice := range splitedDataPoint {

                if(strings.Contains(slice, oldslice)){
                        r := strings.NewReplacer(oldslice + "_","")
                        slice = r.Replace(slice)
                }

                oldslice = slice

                if(i>0){ newDatapoint = newDatapoint + "." + slice }

        }


        return newDatapoint
}*/

func perfDump(binary string, socket *socket) (string, error) {
        cmdArgs := []string{"--admin-daemon", socket.socket}
        if socket.sockType == typeOsd {
                cmdArgs = append(cmdArgs, "perf", "dump")
        } else if socket.sockType == typeMon {
                cmdArgs = append(cmdArgs, "perfcounters_dump")
        } else {
                return "", fmt.Errorf("[Unknown socket type] %s", socket.sockType)
        }


        cmd := exec.Command(binary, cmdArgs...)

        var out bytes.Buffer
        cmd.Stdout = &out
        var stderr bytes.Buffer
        cmd.Stderr = &stderr

        err := cmd.Run()
        if err != nil {
                return "", fmt.Errorf("[Error running ceph dump command] %s", stderr.String())
        }
        return out.String(), nil

}

func parseDump(dump string) (map[string]interface{}, error){
        data := make(map[string]interface{})

        err := json.Unmarshal([]byte(dump), &data)

        return data, err

}
