package self

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
)

type LeaderInfo struct {
	Leader    string `json:"leader"`
	StartTime string `json:"startTime"`
	Uptime    string `json:"uptime"`
}

type AppendRequest struct {
	Count int64 `json:"recvAppendRequestCnt"`
}

type Recv struct {
	Appendrequest AppendRequest
	Bandwithrate  float64 `json:"recvBandwithRate"`
	Pkgrate       float64 `json:"recvPkgRate"`
}

type sendAppendRequest struct {
	Cnt int64 `json:"sendAppendRequestCnt"`
}

type Send struct {
	AppendRequest sendAppendRequest
	BandwithRate  float64 `json:"sendBandwidthRate"`
	PkgRate       float64 `json:"sendPkgRate"`
}

type Self struct {
	ID         string `json:"id"`
	LeaderInfo LeaderInfo
	Name       string `json:"name"`
	Recv       Recv
	Send       Send
	StartTime  string `json:"startTime"`
	State      string `json:"state"`
}

func eventMapping(content []byte) common.MapStr {
	var data Self
	json.Unmarshal(content, &data)
	event := common.MapStr{
		"id": data.ID,
		"leaderinfo": common.MapStr{
			"leader":    data.LeaderInfo.Leader,
			"starttime": data.LeaderInfo.StartTime,
			"uptime":    data.LeaderInfo.Uptime,
		},
		"name": data.Name,
		"recv": common.MapStr{
			"appendrequest": common.MapStr{
				"count": data.Recv.Appendrequest.Count,
			},
			"bandwithrate": data.Recv.Bandwithrate,
			"pkgrate":      data.Recv.Pkgrate,
		},
		"send": common.MapStr{
			"appendrequest": common.MapStr{
				"count": data.Send.AppendRequest.Cnt,
			},
			"bandwithrate": data.Send.BandwithRate,
			"pkgrate":      data.Send.PkgRate,
		},
		"starttime": data.StartTime,
		"state":     data.State,
	}

	return event
}
