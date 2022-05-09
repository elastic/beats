package api

import (
	context "context"
)

func NewProducerMock(cap int) *ProducerMock {
	return &ProducerMock{
		Q: make([]*Event, 0, cap),
	}
}

type ProducerMock struct {
	UnimplementedProducerServer
	Q []*Event
}

func (p *ProducerMock) PublishEvents(ctx context.Context, r *PublishRequest) (*PublishReply, error) {
	resp := &PublishReply{
		Results: make([]*EventResult, 0, len(r.Events)),
	}

	for _, e := range r.Events {
		if len(p.Q) == cap(p.Q) {
			return resp, nil
		}

		p.Q = append(p.Q, e)

		resp.Results = append(resp.Results, &EventResult{
			Timestamp: e.GetTimestamp(),
			QueueId:   "queue",
			EventId:   e.GetEventId(),
		})
	}

	return resp, nil
}
