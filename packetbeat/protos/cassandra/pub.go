package cassandra

import (
	"fmt"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/publish"
)

// Transaction Publisher.
type transPub struct {
	sendRequest        bool
	sendResponse       bool
	sendRequestHeader  bool
	sendResponseHeader bool

	results publish.Transactions
}

func (pub *transPub) onTransaction(requ, resp *message) error {
	if pub.results == nil {
		return nil
	}

	event := pub.createEvent(requ, resp)
	pub.results.PublishTransaction(event)
	return nil
}

func (pub *transPub) createEvent(requ, resp *message) common.MapStr {
	status := common.OK_STATUS

	if resp.failed {
		status = common.ERROR_STATUS
	}

	// resp_time in milliseconds
	responseTime := int32(resp.Ts.Sub(requ.Ts).Nanoseconds() / 1e6)

	src := &common.Endpoint{
		Ip:   requ.Tuple.Src_ip.String(),
		Port: requ.Tuple.Src_port,
		Proc: string(requ.CmdlineTuple.Src),
	}
	dst := &common.Endpoint{
		Ip:   requ.Tuple.Dst_ip.String(),
		Port: requ.Tuple.Dst_port,
		Proc: string(requ.CmdlineTuple.Dst),
	}

	event := common.MapStr{
		"@timestamp":   common.Time(requ.Ts),
		"type":         "cassandra",
		"status":       status,
		"responsetime": responseTime,
		"bytes_in":     requ.Size,
		"bytes_out":    resp.Size,
		"src":          src,
		"dst":          dst,
	}

	// add processing notes/errors to event
	if len(requ.Notes)+len(resp.Notes) > 0 {
		event["notes"] = append(requ.Notes, resp.Notes...)
	}

	if pub.sendRequest {
		if pub.sendRequestHeader {
			requ.data["request_headers"] = requ.header.toMap()
		}

		event["cassandra_request"] = requ.data
	}

	if pub.sendResponse {
		if pub.sendResponseHeader {
			resp.data["response_headers"] = resp.header.toMap()
		}

		event["cassandra_response"] = resp.data
	}

	if logp.IsDebug("cassandra") {
		logp.Debug("cassandra", fmt.Sprint(event))
	}

	return event
}
