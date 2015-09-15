package icmp

import "time"

type icmpTransaction struct {
	Ts    time.Time // timestamp of the first packet
	Tuple icmpTuple
	Notes []string

	Request  *icmpMessage
	Response *icmpMessage
}

func (t *icmpTransaction) HasError() bool {
	return t.Request == nil ||
		(t.Request != nil && isError(&t.Tuple, t.Request)) ||
		(t.Response != nil && isError(&t.Tuple, t.Response)) ||
		(t.Request != nil && t.Response == nil && requiresCounterpart(&t.Tuple, t.Request))
}

func (t *icmpTransaction) ResponseTimeMillis() (int32, bool) {
	if t.Request != nil && t.Response != nil {
		return int32(t.Response.Ts.Sub(t.Request.Ts).Nanoseconds() / 1e6), true
	} else {
		return 0, false
	}
}
