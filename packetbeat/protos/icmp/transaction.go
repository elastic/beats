package icmp

import "time"

type icmpTransaction struct {
	ts    time.Time // timestamp of the first packet
	tuple icmpTuple
	notes []string

	request  *icmpMessage
	response *icmpMessage
}

func (t *icmpTransaction) HasError() bool {
	return t.request == nil ||
		(t.request != nil && isError(&t.tuple, t.request)) ||
		(t.response != nil && isError(&t.tuple, t.response)) ||
		(t.request != nil && t.response == nil && requiresCounterpart(&t.tuple, t.request))
}

func (t *icmpTransaction) ResponseTimeMillis() (int32, bool) {
	if t.request != nil && t.response != nil {
		return int32(t.response.ts.Sub(t.request.ts).Nanoseconds() / 1e6), true
	}
	return 0, false
}
