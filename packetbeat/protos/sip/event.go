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

package sip

import (
	"github.com/elastic/beats/v8/libbeat/common"
)

// ProtocolFields contains SIP fields.
type ProtocolFields struct {
	Code    int              `ecs:"code"`
	Method  common.NetString `ecs:"method"`
	Status  common.NetString `ecs:"status"`
	Type    string           `ecs:"type"`
	Version string           `ecs:"version"`

	URIOriginal common.NetString `ecs:"uri.original"`
	URIScheme   common.NetString `ecs:"uri.scheme"`
	URIUsername common.NetString `ecs:"uri.username"`
	URIHost     common.NetString `ecs:"uri.host"`
	URIPort     int              `ecs:"uri.port"`

	Accept            common.NetString `ecs:"accept"`
	Allow             []string         `ecs:"allow"`
	CallID            common.NetString `ecs:"call_id"`
	ContentLength     int              `ecs:"content_length"`
	ContentType       common.NetString `ecs:"content_type"`
	MaxForwards       int              `ecs:"max_forwards"`
	Supported         []string         `ecs:"supported"`
	UserAgentOriginal common.NetString `ecs:"user_agent.original"`

	PrivateURIOriginal common.NetString `ecs:"private.uri.original"`
	PrivateURIScheme   common.NetString `ecs:"private.uri.scheme"`
	PrivateURIUsername common.NetString `ecs:"private.uri.username"`
	PrivateURIHost     common.NetString `ecs:"private.uri.host"`
	PrivateURIPort     int              `ecs:"private.uri.port"`

	CseqCode   int              `ecs:"cseq.code"`
	CseqMethod common.NetString `ecs:"cseq.method"`

	ViaOriginal []common.NetString `ecs:"via.original"`

	ToDisplayInfo common.NetString `ecs:"to.display_info"`
	ToURIOriginal common.NetString `ecs:"to.uri.original"`
	ToURIScheme   common.NetString `ecs:"to.uri.scheme"`
	ToURIUsername common.NetString `ecs:"to.uri.username"`
	ToURIHost     common.NetString `ecs:"to.uri.host"`
	ToURIPort     int              `ecs:"to.uri.port"`
	ToTag         common.NetString `ecs:"to.tag"`

	FromDisplayInfo common.NetString `ecs:"from.display_info"`
	FromURIOriginal common.NetString `ecs:"from.uri.original"`
	FromURIScheme   common.NetString `ecs:"from.uri.scheme"`
	FromURIUsername common.NetString `ecs:"from.uri.username"`
	FromURIHost     common.NetString `ecs:"from.uri.host"`
	FromURIPort     int              `ecs:"from.uri.port"`
	FromTag         common.NetString `ecs:"from.tag"`

	ContactDisplayInfo common.NetString `ecs:"contact.display_info"`
	ContactURIOriginal common.NetString `ecs:"contact.uri.original"`
	ContactURIScheme   common.NetString `ecs:"contact.uri.scheme"`
	ContactURIUsername common.NetString `ecs:"contact.uri.username"`
	ContactURIHost     common.NetString `ecs:"contact.uri.host"`
	ContactURIPort     int              `ecs:"contact.uri.port"`
	ContactTransport   common.NetString `ecs:"contact.transport"`
	ContactLine        common.NetString `ecs:"contact.line"`
	ContactExpires     int              `ecs:"contact.expires"`
	ContactQ           float64          `ecs:"contact.q"`

	AuthScheme      common.NetString `ecs:"auth.scheme"`
	AuthRealm       common.NetString `ecs:"auth.realm"`
	AuthURIOriginal common.NetString `ecs:"auth.uri.original"`
	AuthURIScheme   common.NetString `ecs:"auth.uri.scheme"`
	AuthURIHost     common.NetString `ecs:"auth.uri.host"`
	AuthURIPort     int              `ecs:"auth.uri.port"`

	SDPVersion       string           `ecs:"sdp.version"`
	SDPOwnerUsername common.NetString `ecs:"sdp.owner.username"`
	SDPOwnerSessID   common.NetString `ecs:"sdp.owner.session_id"`
	SDPOwnerVersion  common.NetString `ecs:"sdp.owner.version"`
	SDPOwnerIP       common.NetString `ecs:"sdp.owner.ip"`
	SDPSessName      common.NetString `ecs:"sdp.session.name"`
	SDPConnInfo      common.NetString `ecs:"sdp.connection.info"`
	SDPConnAddr      common.NetString `ecs:"sdp.connection.address"`
	SDPBodyOriginal  common.NetString `ecs:"sdp.body.original"`
}
