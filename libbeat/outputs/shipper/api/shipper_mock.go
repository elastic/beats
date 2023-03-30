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
	"errors"
	"time"

	pb "github.com/elastic/elastic-agent-shipper-client/pkg/proto"
	"github.com/elastic/elastic-agent-shipper-client/pkg/proto/messages"

	"github.com/gofrs/uuid"
)

func NewProducerMock(cap int) *ProducerMock {
	id, _ := uuid.NewV4()
	return &ProducerMock{
		uuid: id.String(),
		Q:    make([]*messages.Event, 0, cap),
	}
}

type ProducerMock struct {
	pb.UnimplementedProducerServer
	Q              []*messages.Event
	uuid           string
	AcceptedCount  uint32
	persistedIndex uint64
	ErrorCallback  func(events []*messages.Event) error
}

func (p *ProducerMock) PublishEvents(ctx context.Context, r *messages.PublishRequest) (*messages.PublishReply, error) {
	if p.ErrorCallback != nil {
		if err := p.ErrorCallback(r.Events); err != nil {
			return nil, err
		}
	}

	if r.Uuid != p.uuid {
		return nil, errors.New("UUID does not match")
	}

	resp := &messages.PublishReply{}

	for _, e := range r.Events {
		if len(p.Q) == cap(p.Q) {
			return resp, nil
		}

		p.Q = append(p.Q, e)
		resp.AcceptedCount++
		if resp.AcceptedCount == p.AcceptedCount {
			break
		}
	}

	resp.AcceptedIndex = uint64(len(p.Q))

	return resp, nil
}

func (p *ProducerMock) Persist(count uint64) {
	p.persistedIndex = count
}

func (p *ProducerMock) PersistedIndex(req *messages.PersistedIndexRequest, producer pb.Producer_PersistedIndexServer) error {
	err := producer.Send(&messages.PersistedIndexReply{
		Uuid:           p.uuid,
		PersistedIndex: p.persistedIndex,
	})
	if err != nil {
		return err
	}

	if !req.PollingInterval.IsValid() || req.PollingInterval.AsDuration() == 0 {
		return nil
	}

	ticker := time.NewTicker(req.PollingInterval.AsDuration())
	defer ticker.Stop()

	for range ticker.C {
		err = producer.Send(&messages.PersistedIndexReply{
			Uuid:           p.uuid,
			PersistedIndex: p.persistedIndex,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
