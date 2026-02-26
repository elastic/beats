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

func (nfs *nfs) getRequestInfo(xdr *xdr) (mapstr.M, error) {
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
			tag, err := xdr.getDynamicOpaque()
			if err != nil {
				return nil, err
			}
			nfsInfo["tag"] = string(tag)
			minor, err := xdr.getUInt()
			if err != nil {
				return nil, err
			}
			nfsInfo["minor_version"] = minor
			opcode, err := nfs.findV4MainOpcode(xdr)
			if err != nil {
				return nil, err
			}
			nfsInfo["opcode"] = opcode
		}
	}
	return nfsInfo, nil
}

func (nfs *nfs) getNFSReplyStatus(xdr *xdr) (string, error) {
	switch nfs.proc {
	case 0:
		return nfsStatus[0], nil
	default:
		stat, err := xdr.getUInt()
		if err != nil {
			return "", err
		}
		return nfsStatus[int(stat)], nil
	}
}
