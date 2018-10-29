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
	"fmt"
	"strconv"
	"syscall"

	"github.com/joeshaw/multierror"
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

type portToServiceMap map[uint32]RpcService

type RpcPortmap struct {
	portmap     portToServiceMap
	rpcBindAddr C.struct_sockaddr_in
}

func (service RpcService) toMapStr() common.MapStr {
	return common.MapStr{
		"program.number": service.programNumber,
		"vers":           service.vers,
		"protocol":       service.protocol,
		"port":           service.port,
		"program.name":   service.programName,
	}
}

func NewRpcPortmap() (*RpcPortmap, error) {
	addr := C.struct_sockaddr_in{}

	// By default, get_myaddress will set the loopback address (INADDR_LOOPBACK)
	// and port 111 (PMAPPORT in pmap_prot.h)
	_, err := C.get_myaddress(&addr)
	if err != nil {
		return nil, errors.Wrap(err, "error getting address for RPC")
	}

	return &RpcPortmap{
		rpcBindAddr: addr,
	}, nil
}

func (portmap *RpcPortmap) Lookup(port uint32) *RpcService {
	service, found := portmap.portmap[port]
	if found {
		return &service
	} else {
		return nil
	}
}

func (portmap *RpcPortmap) Refresh() error {
	portmap.portmap = make(portToServiceMap, len(portmap.portmap))
	err := portmap.loadRpcPortmap()
	return err
}

func (portmap *RpcPortmap) RpcPort() uint32 {
	return uint32(portmap.rpcBindAddr.sin_port)
}

// loadRpcPortmap retrieves a map of ports to RPC services using pmap_getmaps(3).
func (portmap *RpcPortmap) loadRpcPortmap() error {
	/*
		struct pmaplist {
			struct pmap pml_map;
			struct pmaplist *pml_next;
		}
	*/
	pmaplist, err := C.pmap_getmaps(&portmap.rpcBindAddr)
	if err != nil {
		return errors.Wrap(err, "error getting RPC port maps")
	}

	var errs multierror.Errors
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
			return errors.Wrap(err, "error getting RPC program name")
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

		existingService, found := portmap.portmap[service.port]
		if !found {
			portmap.portmap[service.port] = service
		} else if found && existingService.programName != service.programName {
			errs = append(errs, fmt.Errorf("conflicting RPC program names for port '%d': '%v' and '%v'", service.port,
				existingService.programName, service.programName))
		}
	}

	return errs.Err()
}
