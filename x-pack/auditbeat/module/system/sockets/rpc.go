// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux

package sockets

// #include <rpc/pmap_prot.h>
// #include <rpc/pmap_clnt.h>
// #include <rpc/netdb.h>
import "C"

import (
	"strconv"
	"syscall"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
)

// RpcService represents an RPC service.
type RpcService struct {
	programNumber uint32
	vers          uint32
	protocol      string
	port          uint32
	programName   string
}

type RpcPortMap map[uint32]([]RpcService)

func (service RpcService) toMapStr() common.MapStr {
	return common.MapStr{
		"program.number": service.programNumber,
		"vers":           service.vers,
		"protocol":       service.protocol,
		"port":           service.port,
		"program.name":   service.programName,
	}
}

// GetRpcPortmap retrieves a map of ports to RPC services using pmap_getmaps(3).
func GetRpcPortmap() (*RpcPortMap, error) {
	addr := C.struct_sockaddr_in{}

	// By default, get_myaddress will set the loopback address (INADDR_LOOPBACK)
	// and port 111 (PMAPPORT in pmap_prot.h)
	_, err := C.get_myaddress(&addr)
	if err != nil {
		return nil, errors.Wrap(err, "error getting address for RPC")
	}

	/*
		struct pmaplist {
			struct pmap pml_map;
			struct pmaplist *pml_next;
		}
	*/
	pmaplist, err := C.pmap_getmaps(&addr)
	if err != nil {
		return nil, errors.Wrap(err, "error getting RPC port maps")
	}

	portmap := make(RpcPortMap)
	for ; pmaplist != nil; pmaplist = pmaplist.pml_next {
		/*
			struct pmap {
				long unsigned pm_prog;
				long unsigned pm_vers;
				long unsigned pm_prot;
				long unsigned pm_port;
			};
		*/
		service := RpcService{
			programNumber: uint32(pmaplist.pml_map.pm_prog),
			vers:          uint32(pmaplist.pml_map.pm_vers),
			port:          uint32(pmaplist.pml_map.pm_port),
		}

		/*
			struct rpcent {
				char *r_name;         // Name of server for this rpc program.
				char **r_aliases;     // Alias list.
				int r_number;         // RPC program number.
			};
		*/
		rpcent, err := C.getrpcbynumber(C.int(pmaplist.pml_map.pm_prog))
		if err != nil {
			return nil, errors.Wrap(err, "error getting RPC program name")
		}
		service.programName = C.GoString(rpcent.r_name)

		switch protocol := uint64(pmaplist.pml_map.pm_prot); protocol {
		case syscall.IPPROTO_UDP:
			service.protocol = "udp"
		case syscall.IPPROTO_TCP:
			service.protocol = "tcp"
		default:
			service.protocol = strconv.FormatUint(protocol, 10)
		}

		portmap[service.port] = append(portmap[service.port], service)
	}

	return &portmap, nil
}
