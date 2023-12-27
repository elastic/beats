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

package nfs

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/packetbeat/pb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type nfs struct {
	vers  uint32
	proc  uint32
	pbf   *pb.Fields
	event beat.Event
}

func (nfs *nfs) getRequestInfo(xdr *xdr) mapstr.M {
	nfsInfo := mapstr.M{
		"version": nfs.vers,
	}

	switch nfs.vers {
	case 3:
		nfsInfo["opcode"] = nfs.getV3Opcode(int(nfs.proc))
	case 4:
		switch nfs.proc {
		case 0:
			nfsInfo["opcode"] = "NULL"
		case 1:
			tag := xdr.getDynamicOpaque()
			nfsInfo["tag"] = string(tag)
			nfsInfo["minor_version"] = xdr.getUInt()
			nfsInfo["opcode"] = nfs.findV4MainOpcode(xdr)
		}
	}
	return nfsInfo
}

func (nfs *nfs) getNFSReplyStatus(xdr *xdr) string {
	switch nfs.proc {
	case 0:
		return nfsStatus[0]
	default:
		stat := int(xdr.getUInt())
		return nfsStatus[stat]
	}
}
