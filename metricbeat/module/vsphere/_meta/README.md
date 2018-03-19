# Testing using GOVCSIM.


To test the vsphere module without a real Vmware SDK URL you can use the following setup. Govcsim is a vCenter Server and ESXi API based simulator written using govmomi. It creates a vCenter Server model with a datacenter, hosts, cluster, resource pools, networks and a datastore.


Requirements:
- golang 1.7+ installed on a system
- git installed on a system

1. Set the GOPATH where govcsim will be installed
```
export GOPATH=/directory/code
```

2. Install Govcsim
```
go get -u github.com/vmware/vic/cmd/vcsim
```

3. Run Govcsim
```
$GOPATH/bin/vcsim
```

Now setup your metricbeat config to connect to Govcsim:

```
- module: vsphere
  metricsets:
    - datastore
    - host
    - virtualmachine
  enabled: true
  period: 5s
  hosts: ["https://127.0.0.1:8989/sdk"]

  username: "user"
  password: "pass"
  insecure: true
 
```
