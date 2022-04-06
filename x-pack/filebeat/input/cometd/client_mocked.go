// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import (
	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

type mockedConnector struct {
	connectWithError error
	outlet           channel.Outleter
}

var _ channel.Connector = new(mockedConnector)

func (m *mockedConnector) Connect(c *common.Config) (channel.Outleter, error) {
	return m.ConnectWith(c, beat.ClientConfig{})
}

func (m *mockedConnector) ConnectWith(*common.Config, beat.ClientConfig) (channel.Outleter, error) {
	if m.connectWithError != nil {
		return nil, m.connectWithError
	}
	return m.outlet, nil
}

type mockedOutleter struct {
	onEventHandler func(event beat.Event) bool
}

var _ channel.Outleter = new(mockedOutleter)

func (m mockedOutleter) Close() error {
	panic("implement me")
}

func (m mockedOutleter) Done() <-chan struct{} {
	panic("implement me")
}

func (m mockedOutleter) OnEvent(event beat.Event) bool {
	return m.onEventHandler(event)
}
