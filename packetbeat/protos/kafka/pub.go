package kafka

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/publish"
)

// Transaction Publisher.
type transPub struct {
	results publish.Transactions
}

func (pub *transPub) onTransaction(
	requMsg *requestMessage,
	respMsg *responseMessage,
	event common.MapStr,
) error {
	if pub.results == nil {
		return nil
	}

	pub.annotate(requMsg, respMsg, event)
	pub.results.PublishTransaction(event)
	return nil
}

func (pub *transPub) annotate(
	requMsg *requestMessage,
	respMsg *responseMessage,
	event common.MapStr,
) {
	addrTuple := common.IPPortTuple{
		SrcIP:   requMsg.endpoint.IP,
		SrcPort: requMsg.endpoint.Port,
		DstIP:   respMsg.endpoint.IP,
		DstPort: respMsg.endpoint.Port,
	}
	cmdLine := procs.ProcWatcher.FindProcessesTuple(&addrTuple)

	responseTime := int32(respMsg.ts.Sub(requMsg.ts).Nanoseconds() / 1e6)
	src := &common.Endpoint{
		IP:      requMsg.endpoint.IP.String(),
		Port:    requMsg.endpoint.Port,
		Cmdline: string(cmdLine.Src),
	}
	dst := &common.Endpoint{
		IP:      respMsg.endpoint.IP.String(),
		Port:    respMsg.endpoint.Port,
		Cmdline: string(cmdLine.Dst),
	}

	event["@timestamp"] = common.Time(requMsg.ts)
	event["type"] = "kafka"
	event["responsetime"] = responseTime
	event["bytes_in"] = requMsg.size
	event["bytes_out"] = respMsg.size
	event["src"] = src
	event["dst"] = dst

	event["rpc"] = common.MapStr{
		"api":       requMsg.header.APIKey.String(),
		"version":   requMsg.header.Version,
		"client_id": string(requMsg.header.ClientID),
	}
}
