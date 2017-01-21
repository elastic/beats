package cassandra

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/packetbeat/publish"
)

// Transaction Publisher.
type transPub struct {
	sendRequest        bool
	sendResponse       bool
	sendRequestHeader  bool
	sendResponseHeader bool
	ignoredOps         string

	results publish.Transactions
}

func (pub *transPub) onTransaction(requ, resp *message) error {
	if pub.results == nil {
		return nil
	}

	event := pub.createEvent(requ, resp)
	if event != nil {
		pub.results.PublishTransaction(event)
	}
	return nil
}

func (pub *transPub) createEvent(requ, resp *message) common.MapStr {
	status := common.OK_STATUS

	if resp.failed {
		status = common.ERROR_STATUS
	}

	//ignore
	if (resp != nil && resp.ignored) || (requ != nil && requ.ignored) {
		return nil
	}

	event := common.MapStr{
		"type":      "cassandra",
		"status":    status,
		"cassandra": common.MapStr{},
	}

	//requ can be null, if the message is a PUSHed message
	if requ != nil {
		// resp_time in milliseconds
		responseTime := int32(resp.Ts.Sub(requ.Ts).Nanoseconds() / 1e6)

		src := &common.Endpoint{
			IP:   requ.Tuple.SrcIP.String(),
			Port: requ.Tuple.SrcPort,
			Proc: string(requ.CmdlineTuple.Src),
		}

		event["@timestamp"] = common.Time(requ.Ts)
		event["responsetime"] = responseTime
		event["bytes_in"] = requ.Size
		event["src"] = src

		// add processing notes/errors to event
		if len(requ.Notes)+len(resp.Notes) > 0 {
			event["notes"] = append(requ.Notes, resp.Notes...)
		}

		if pub.sendRequest {
			if pub.sendRequestHeader {
				if requ.data == nil {
					requ.data = map[string]interface{}{}
				}
				requ.data["headers"] = requ.header
			}

			if len(requ.data) > 0 {
				event["cassandra"].(common.MapStr)["request"] = requ.data
			}
		}

		dst := &common.Endpoint{
			IP:   requ.Tuple.DstIP.String(),
			Port: requ.Tuple.DstPort,
			Proc: string(requ.CmdlineTuple.Dst),
		}
		event["dst"] = dst

	} else {
		//dealing with PUSH message
		event["no_request"] = true
		event["@timestamp"] = common.Time(resp.Ts)

		dst := &common.Endpoint{
			IP:   resp.Tuple.DstIP.String(),
			Port: resp.Tuple.DstPort,
			Proc: string(resp.CmdlineTuple.Dst),
		}
		event["dst"] = dst
	}

	event["bytes_out"] = resp.Size

	if pub.sendResponse {

		if pub.sendResponseHeader {
			if resp.data == nil {
				resp.data = map[string]interface{}{}
			}

			resp.data["headers"] = resp.header
		}

		if len(resp.data) > 0 {
			event["cassandra"].(common.MapStr)["response"] = resp.data
		}

	}

	return event
}
