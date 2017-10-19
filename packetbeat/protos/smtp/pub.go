package smtp

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/packetbeat/protos"
)

// Transaction Publisher.
type transPub struct {
	sendRequest     bool
	sendResponse    bool
	sendDataHeaders bool
	sendDataBody    bool

	results protos.Reporter
}

func (pub *transPub) onTransaction(requ, resp *message) error {
	if pub.results == nil {
		return nil
	}

	pub.results(pub.createEvent(requ, resp))
	return nil
}

func (pub *transPub) createEvent(requ, resp *message) beat.Event {
	status := common.OK_STATUS
	if resp.statusCode >= 400 {
		status = common.ERROR_STATUS
	}

	fields := common.MapStr{
		"type":      "smtp",
		"status":    status,
		"bytes_out": resp.Size,
	}

	if pub.sendResponse {
		fields["response"] = common.NetString(resp.raw)
	}

	details := common.MapStr{}

	// Some transactions can have no request
	requNotes := []string{}
	ts := resp.Ts
	if requ != nil {
		// resp_time in milliseconds
		responseTime := int32(resp.Ts.Sub(requ.Ts).Nanoseconds() / 1e6)

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

		fields["responsetime"] = responseTime
		fields["src"] = src
		fields["dst"] = dst
		fields["bytes_in"] = requ.Size

		if pub.sendRequest {
			fields["request"] = common.NetString(requ.raw)
		}

		requNotes = requ.Notes
		ts = requ.Ts

		details["request"] = common.MapStr{}
		dr := details["request"].(common.MapStr)

		dr["command"] = requ.command
		if len(requ.param) > 0 {
			dr["param"] = requ.param
		}
		if pub.sendDataHeaders && len(requ.headers) > 0 {
			dr["headers"] = requ.headers
		}
		if pub.sendDataBody && len(requ.body) > 0 {
			dr["body"] = requ.body
		}
	}

	// add processing notes/errors to event
	if len(requNotes)+len(resp.Notes) > 0 {
		fields["notes"] = append(requNotes, resp.Notes...)
	}

	details["response"] = common.MapStr{}
	dr := details["response"].(common.MapStr)
	dr["code"] = resp.statusCode
	if len(resp.statusPhrases) > 0 {
		dr["phrases"] = resp.statusPhrases
	}

	if len(details) > 0 {
		fields["smtp"] = details
	}

	return beat.Event{
		Timestamp: ts,
		Fields:    fields,
	}
}
