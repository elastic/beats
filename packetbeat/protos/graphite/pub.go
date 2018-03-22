package graphite

import (
	"strconv"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/packetbeat/publish"
)

// Transaction Publisher.
type transPub struct {
	sendRequest  bool
	sendResponse bool

	results publish.Transactions
}

func (pub *transPub) onTransaction(requ, resp *message) error {
	if pub.results == nil {
		return nil
	}
	// Generates one event for each metric in pickle 8
	event := pub.createEvent(requ, resp)
	pub.results.PublishTransaction(event)
	return nil
}

func (pub *transPub) createEvent(requ, resp *message) common.MapStr {
	status := common.OK_STATUS

	src := &common.Endpoint{
		IP:   requ.Tuple.SrcIP.String(),
		Port: requ.Tuple.SrcPort,
		Proc: string(requ.CmdlineTuple.Src),
	}
	dst := &common.Endpoint{
		IP:   requ.Tuple.DstIP.String(),
		Port: requ.Tuple.DstPort,
		Proc: string(requ.CmdlineTuple.Dst),
	}
	// To generate one event with all the records in pickle (Array of metrics returned)
	type requestJSON []*JSON
	var list requestJSON
	var timeStamp int64
	var mValue float64
	if len(requ.Notes) == 3 {
		timeStamp, _ = strconv.ParseInt(requ.Notes[2], 10, 64)
		mValue, _ = strconv.ParseFloat(requ.Notes[1], 64)
		list = append(list, &JSON{
			MetricName:      requ.Notes[0],
			MetricValue:     mValue,
			MetricTimestamp: timeStamp,
		})
	} else {
		for k := 0; k < len(requ.Notes); k = k + 3 {
			timeStamp, _ = strconv.ParseInt(requ.Notes[k+1], 10, 64)
			mValue, _ = strconv.ParseFloat(requ.Notes[k+2], 64)
			list = append(list, &JSON{
				MetricName:      requ.Notes[k],
				MetricValue:     mValue,
				MetricTimestamp: timeStamp,
			})
		}
	}

	event := common.MapStr{
		"@timestamp":   common.Time(requ.Ts),
		"type":         "graphite",
		"status":       status,
		"responsetime": 0,
		"bytes_in":     requ.Size,
		"bytes_out":    0,
		"src":          src,
		"dst":          dst,
		"request":      list,
	}

	return event
}
