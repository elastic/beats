# govcsim - A vCenter and ESXi API based simulator

This package implements a vSphere Web Services (SOAP) SDK endpoint intended for testing consumers of the API.
While the package is written in the Go language, it can be used by any language that can talk to the vSphere API.

## Installation

Note: This package is currently being incubated with the VIC repository, but at some point will be moved to the govmomi
repository.

```
% export GOPATH=$HOME/gopath
% go get -u github.com/vmware/vic/cmd/vcsim
% $GOPATH/bin/vcsim -h
```

## Usage

The **vcsim** program by default creates a *vCenter* model with a datacenter, hosts, cluster, resource pools, networks
and a datastore.  The naming is similar to that of the original *vcsim* mode that was included with vCenter.  The number
of resources can be increased or decreased using the various resource type flags.  Resources can also be created and
removed using the API.

Example using the default settings:

```
% export GOVC_URL=https://user:pass@127.0.0.1:8989
% $GOPATH/vcsim
% govc find
/
/DC0
/DC0/vm
/DC0/vm/DC0_H0_VM0
/DC0/vm/DC0_H0_VM1
/DC0/vm/DC0_C0_RP0_VM0
/DC0/vm/DC0_C0_RP0_VM1
/DC0/host
/DC0/host/DC0_H0
/DC0/host/DC0_H0/DC0_H0
/DC0/host/DC0_H0/Resources
/DC0/host/DC0_C0
/DC0/host/DC0_C0/DC0_C0_H0
/DC0/host/DC0_C0/DC0_C0_H1
/DC0/host/DC0_C0/DC0_C0_H2
/DC0/host/DC0_C0/Resources
/DC0/datastore
/DC0/datastore/LocalDS_0
/DC0/network
/DC0/network/VM Network
/DC0/network/DVS0
/DC0/network/DC0_DVPG0
```

Example using ESX mode:

```
% $GOPATH/vcsim -esx
% govc find
/
/ha-datacenter
/ha-datacenter/vm
/ha-datacenter/vm/ha-host_VM0
/ha-datacenter/vm/ha-host_VM1
/ha-datacenter/host
/ha-datacenter/host/localhost.localdomain
/ha-datacenter/host/localhost.localdomain/localhost.localdomain
/ha-datacenter/host/localhost.localdomain/Resources
/ha-datacenter/datastore
/ha-datacenter/datastore/LocalDS_0
/ha-datacenter/network
/ha-datacenter/network/VM Network

```

## Supported methods

The simulator supports a subset of API methods.  However, the generated [govmomi](https://github.com/vmware/govmomi)
code includes all types and methods defined in the vmodl, which can be used to implement any method documented in the
[VMware vSphere API Reference](http://pubs.vmware.com/vsphere-65/index.jsp#com.vmware.wssdk.apiref.doc/right-pane.html).

To see the list of supported methods:

```
curl -k https://user:pass@127.0.0.1:8989/about
```
