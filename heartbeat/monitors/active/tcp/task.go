package tcp

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/transport"

	"github.com/elastic/beats/heartbeat/look"
	"github.com/elastic/beats/heartbeat/reason"
)

func pingHost(
	dialer transport.Dialer,
	host string,
	timeout time.Duration,
	validator ConnCheck,
) (common.MapStr, error) {
	start := time.Now()
	deadline := start.Add(timeout)

	conn, err := dialer.Dial("tcp", host)
	if err != nil {
		debugf("dial failed with: %v", err)
		return nil, reason.IOFailed(err)
	}
	defer conn.Close()
	if validator == nil {
		// no additional validation step => ping success
		return common.MapStr{}, nil
	}

	if err := conn.SetDeadline(deadline); err != nil {
		debugf("setting connection deadline failed with: %v", err)
		return nil, reason.IOFailed(err)
	}

	validateStart := time.Now()
	err = validator.Validate(conn)
	if err != nil && err != errRecvMismatch {
		debugf("check failed with: %v", err)
		return nil, reason.IOFailed(err)
	}

	end := time.Now()
	event := common.MapStr{
		"tcp": common.MapStr{
			"rtt": common.MapStr{
				"validate": look.RTT(end.Sub(validateStart)),
			},
		},
	}
	if err != nil {
		event["error"] = reason.FailValidate(err)
	}
	return event, nil
}
