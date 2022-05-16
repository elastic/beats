// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
	Q     []*Event
	Error error
}

func (p *ProducerMock) PublishEvents(ctx context.Context, r *PublishRequest) (*PublishReply, error) {
	if p.Error != nil {
		return nil, p.Error
	}

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
