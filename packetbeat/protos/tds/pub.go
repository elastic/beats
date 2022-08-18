package tds

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"

	"github.com/elastic/beats/v7/packetbeat/pb"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Transaction Publisher.
type transPub struct {
	sendRequest  bool
	sendResponse bool

	results protos.Reporter
}

func (pub *transPub) onTransaction(requ, resp *message) error {
	logp.Info("pub.onTransaction()")
	if pub.results == nil {
		return nil
	}

	pub.results(pub.createEvent(requ, resp))
	return nil
}

func (pub *transPub) createEvent(requ, resp *message) beat.Event {
	logp.Info("pub.createEvent()")

	status := common.OK_STATUS

	// resp_time in millisecond
	responseTime := int32(resp.Ts.Sub(requ.Ts).Nanoseconds() / 1e6)

	src := &common.Endpoint{
		IP:      requ.Tuple.SrcIP.String(),
		Port:    requ.Tuple.SrcPort,
		Process: requ.CmdlineTuple.Src,
	}

	dst := &common.Endpoint{
		IP:      requ.Tuple.DstIP.String(),
		Port:    requ.Tuple.DstPort,
		Process: requ.CmdlineTuple.Dst,
	}

	if pub.sendRequest {
		// fields["request"] =
	}

	if pub.sendResponse {
		// fields["response"] =
	}

	evt, pbf := pb.NewBeatEvent(requ.Ts)
	fields := evt.Fields
	fields["type"] = "tds"
	fields["status"] = status
	fields["responsetime"] = responseTime
	fields["bytes_in"] = requ.Size
	fields["bytes_out"] = resp.Size
	fields["src"] = src
	fields["dst"] = dst
	fields["requestType"] = requ.requestType
	// add processing notes/errors to event
	if len(requ.Notes)+len(resp.Notes) > 0 {
		fields["notes"] = append(requ.Notes, resp.Notes...)
	}
	pbf.SetSource(src)
	pbf.SetDestination(dst)
	evt.Fields = fields
	pbf.Network.Transport = "tcp"
	pbf.Network.Protocol = pbf.Event.Dataset

	logp.Info("** event: %v", evt)
	return evt
}
