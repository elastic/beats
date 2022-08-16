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

	pb "github.com/elastic/elastic-agent-shipper-client/pkg/proto"
	"github.com/elastic/elastic-agent-shipper-client/pkg/proto/messages"

	"github.com/gofrs/uuid"
)

func NewProducerMock(cap int) *ProducerMock {
	id, _ := uuid.NewV4()
	return &ProducerMock{
		UUID: id.String(),
		Q:    make([]*messages.Event, 0, cap),
	}
}

type ProducerMock struct {
	pb.UnimplementedProducerServer
	Q     []*messages.Event
	UUID  string
	Error error
}

func (p *ProducerMock) PublishEvents(ctx context.Context, r *messages.PublishRequest) (*messages.PublishReply, error) {
	if p.Error != nil {
		return nil, p.Error
	}

	resp := &messages.PublishReply{}

	for _, e := range r.Events {
		if len(p.Q) == cap(p.Q) {
			return resp, nil
		}

		p.Q = append(p.Q, e)
		resp.AcceptedCount++
	}

	resp.AcceptedIndex = uint64(len(p.Q))
	if resp.AcceptedIndex > 0 {
		resp.PersistedIndex = resp.AcceptedIndex - 1 // so we trigger the use of `PersistedIndex`
	}

	return resp, nil
}

func (p *ProducerMock) PersistedIndex(req *messages.PersistedIndexRequest, producer pb.Producer_PersistedIndexServer) error {
	return producer.Send(&messages.PersistedIndexReply{
		Uuid:           p.UUID,
		PersistedIndex: uint64(len(p.Q)),
	})
}
