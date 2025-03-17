---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-vsphere.html
---

# vSphere fields [exported-fields-vsphere]

vSphere module


## vsphere [_vsphere]


## cluster [_cluster_4]

Cluster information.

**`vsphere.cluster.datastore.names`**
:   List of all the datastore names associated with the cluster.

type: keyword


**`vsphere.cluster.datastore.count`**
:   Number of datastores associated with the cluster.

type: long


**`vsphere.cluster.das_config.admission.control.enabled`**
:   Indicates whether strict admission control is enabled.

type: boolean


**`vsphere.cluster.das_config.enabled`**
:   Indicates whether vSphere HA feature is enabled.

type: boolean


**`vsphere.cluster.host.count`**
:   Number of hosts associated with the cluster.

type: long


**`vsphere.cluster.host.names`**
:   List of all the host names associated with the cluster.

type: keyword


**`vsphere.cluster.id`**
:   Unique cluster ID.

type: keyword


**`vsphere.cluster.name`**
:   Cluster name.

type: keyword


**`vsphere.cluster.network.count`**
:   Number of networks associated with the cluster.

type: long


**`vsphere.cluster.network.names`**
:   List of all the network names associated with the cluster.

type: keyword


**`vsphere.cluster.triggered_alarms.*`**
:   List of all the triggered alarms.

type: object



## datastore [_datastore]

datastore

**`vsphere.datastore.capacity.free.bytes`**
:   Free bytes of the datastore.

type: long

format: bytes


**`vsphere.datastore.capacity.total.bytes`**
:   Total bytes of the datastore.

type: long

format: bytes


**`vsphere.datastore.capacity.used.bytes`**
:   Used bytes of the datastore.

type: long

format: bytes


**`vsphere.datastore.capacity.used.pct`**
:   Percentage of datastore capacity used.

type: scaled_float

format: percent


**`vsphere.datastore.disk.capacity.bytes`**
:   Configured size of the datastore.

type: long

format: bytes


**`vsphere.datastore.disk.capacity.usage.bytes`**
:   The amount of storage capacity currently being consumed by datastore.

type: long

format: bytes


**`vsphere.datastore.disk.provisioned.bytes`**
:   Amount of storage set aside for use by a datastore.

type: long

format: bytes


**`vsphere.datastore.fstype`**
:   Filesystem type.

type: keyword


**`vsphere.datastore.host.count`**
:   Number of hosts.

type: long


**`vsphere.datastore.host.names`**
:   List of all the host names.

type: keyword


**`vsphere.datastore.id`**
:   Unique datastore ID.

type: keyword


**`vsphere.datastore.name`**
:   Datastore name.

type: keyword


**`vsphere.datastore.read.bytes`**
:   Rate of reading data from the datastore.

type: long

format: bytes


**`vsphere.datastore.status`**
:   Status of the datastore.

type: keyword


**`vsphere.datastore.triggered_alarms.*`**
:   List of all the triggered alarms.

type: object


**`vsphere.datastore.vm.count`**
:   Number of VMs.

type: long


**`vsphere.datastore.vm.names`**
:   List of all the VM names.

type: keyword


**`vsphere.datastore.write.bytes`**
:   Rate of writing data to the datastore.

type: long

format: bytes



## datastorecluster [_datastorecluster]

Datastore Cluster

**`vsphere.datastorecluster.id`**
:   Unique datastore cluster ID.

type: keyword


**`vsphere.datastorecluster.name`**
:   The datastore cluster name.

type: keyword


**`vsphere.datastorecluster.capacity.bytes`**
:   Total capacity of this storage pod, in bytes.

type: long

format: bytes


**`vsphere.datastorecluster.free_space.bytes`**
:   Total free space on this storage pod, in bytes.

type: long

format: bytes


**`vsphere.datastorecluster.datastore.names`**
:   List of all the datastore names associated with the datastore cluster.

type: keyword


**`vsphere.datastorecluster.datastore.count`**
:   Number of datastores in the datastore cluster.

type: long


**`vsphere.datastorecluster.triggered_alarms.*`**
:   List of all the triggered alarms.

type: object



## host [_host_2]

Host information from vSphere environment.

**`vsphere.host.cpu.used.mhz`**
:   Used CPU in MHz.

type: long


**`vsphere.host.cpu.total.mhz`**
:   Total CPU in MHz.

type: long


**`vsphere.host.cpu.free.mhz`**
:   Free CPU in MHz.

type: long


**`vsphere.host.datastore.names`**
:   List of all the datastore names.

type: keyword


**`vsphere.host.datastore.count`**
:   Number of datastores on the host.

type: long


**`vsphere.host.disk.capacity.usage.bytes`**
:   The amount of storage capacity currently being consumed by or on the entity.

type: long

format: bytes


**`vsphere.host.disk.devicelatency.average.ms`**
:   Average amount of time it takes to complete an SCSI command from physical device in milliseconds.

type: long


**`vsphere.host.disk.latency.total.ms`**
:   Highest latency value across all disks used by the host in milliseconds.

type: long


**`vsphere.host.disk.read.bytes`**
:   Average number of bytes read from the disk each second.

type: long

format: bytes


**`vsphere.host.disk.write.bytes`**
:   Average number of bytes written to the disk each second.

type: long

format: bytes


**`vsphere.host.disk.total.bytes`**
:   Sum of disk read and write rates each second in bytes.

type: long

format: bytes


**`vsphere.host.id`**
:   Unique host ID.

type: keyword


**`vsphere.host.memory.free.bytes`**
:   Free Memory in bytes.

type: long

format: bytes


**`vsphere.host.memory.total.bytes`**
:   Total Memory in bytes.

type: long

format: bytes


**`vsphere.host.memory.used.bytes`**
:   Used Memory in bytes.

type: long

format: bytes


**`vsphere.host.name`**
:   Host name.

type: keyword


**`vsphere.host.network_names`**
:   Network names.

type: keyword


**`vsphere.host.network.names`**
:   List of all the network names.

type: keyword


**`vsphere.host.network.count`**
:   Number of networks on the host.

type: long


**`vsphere.host.network.bandwidth.transmitted.bytes`**
:   Average rate at which data was transmitted during the interval. This represents the bandwidth of the network.

type: long

format: bytes


**`vsphere.host.network.bandwidth.received.bytes`**
:   Average rate at which data was received during the interval. This represents the bandwidth of the network.

type: long

format: bytes


**`vsphere.host.network.bandwidth.total.bytes`**
:   Sum of network transmitted and received rates in bytes during the interval.

type: long

format: bytes


**`vsphere.host.network.packets.transmitted.count`**
:   Number of packets transmitted.

type: long


**`vsphere.host.network.packets.received.count`**
:   Number of packets received.

type: long


**`vsphere.host.network.packets.errors.transmitted.count`**
:   Number of packets with errors transmitted.

type: long


**`vsphere.host.network.packets.errors.received.count`**
:   Number of packets with errors received.

type: long


**`vsphere.host.network.packets.errors.total.count`**
:   Total number of packets with errors.

type: long


**`vsphere.host.network.packets.multicast.transmitted.count`**
:   Number of multicast packets transmitted.

type: long


**`vsphere.host.network.packets.multicast.received.count`**
:   Number of multicast packets received.

type: long


**`vsphere.host.network.packets.multicast.total.count`**
:   Total number of multicast packets.

type: long


**`vsphere.host.network.packets.dropped.transmitted.count`**
:   Number of transmitted packets dropped.

type: long


**`vsphere.host.network.packets.dropped.received.count`**
:   Number of received packets dropped.

type: long


**`vsphere.host.network.packets.dropped.total.count`**
:   Total number of packets dropped.

type: long


**`vsphere.host.status`**
:   The overall health status of a host in the vSphere environment.

type: keyword


**`vsphere.host.triggered_alarms.*`**
:   List of all the triggered alarms.

type: object


**`vsphere.host.uptime`**
:   The total uptime of a host in seconds within the vSphere environment.

type: long


**`vsphere.host.vm.names`**
:   List of all the VM names.

type: keyword


**`vsphere.host.vm.count`**
:   Number of virtual machines on the host.

type: long



## network [_network_12]

Network-related information.

**`vsphere.network.accessible`**
:   Indicates whether at least one host is configured to provide this network.

type: boolean


**`vsphere.network.config.status`**
:   Indicates whether the system has detected a configuration issue.

type: keyword


**`vsphere.network.host.names`**
:   Names of the hosts connected to this network.

type: keyword


**`vsphere.network.host.count`**
:   Number of hosts connected to this network.

type: long


**`vsphere.network.id`**
:   Unique network ID.

type: keyword


**`vsphere.network.name`**
:   Name of the network.

type: keyword


**`vsphere.network.status`**
:   General health of the network.

type: keyword


**`vsphere.network.type`**
:   Type of the network (e.g., Network(Standard), DistributedVirtualport).

type: keyword


**`vsphere.network.vm.names`**
:   Names of the virtual machines connected to this network.

type: keyword


**`vsphere.network.vm.count`**
:   Number of virtual machines connected to this network.

type: long


**`vsphere.network.triggered_alarms.*`**
:   List of all the triggered alarms.

type: object



## resourcepool [_resourcepool]

Resource pool information from vSphere environment.

**`vsphere.resourcepool.cpu.usage.mhz`**
:   Basic CPU performance statistics, in MHz.

type: long


**`vsphere.resourcepool.cpu.demand.mhz`**
:   Basic CPU performance statistics, in MHz.

type: long


**`vsphere.resourcepool.cpu.entitlement.mhz`**
:   The amount of CPU resource, in MHz, that this VM is entitled to, as calculated by DRS.

type: long


**`vsphere.resourcepool.cpu.entitlement.static.mhz`**
:   The static CPU resource entitlement for a virtual machine.

type: long


**`vsphere.resourcepool.id`**
:   Unique resource pool ID.

type: keyword


**`vsphere.resourcepool.memory.usage.guest.bytes`**
:   Guest memory utilization statistics, in bytes.

type: long

format: bytes


**`vsphere.resourcepool.memory.usage.host.bytes`**
:   Host memory utilization statistics, in bytes.

type: long

format: bytes


**`vsphere.resourcepool.memory.entitlement.bytes`**
:   The amount of memory, in bytes, that this VM is entitled to, as calculated by DRS.

type: long

format: bytes


**`vsphere.resourcepool.memory.entitlement.static.bytes`**
:   The static memory resource entitlement for a virtual machine, in bytes.

type: long

format: bytes


**`vsphere.resourcepool.memory.private.bytes`**
:   The portion of memory, in bytes, that is granted to a virtual machine from non-shared host memory.

type: long

format: bytes


**`vsphere.resourcepool.memory.shared.bytes`**
:   The portion of memory, in bytes, that is granted to a virtual machine from host memory that is shared between VMs.

type: long

format: bytes


**`vsphere.resourcepool.memory.swapped.bytes`**
:   The portion of memory, in bytes, that is granted to a virtual machine from the host’s swap space.

type: long

format: bytes


**`vsphere.resourcepool.memory.ballooned.bytes`**
:   The size of the balloon driver in a virtual machine, in bytes.

type: long

format: bytes


**`vsphere.resourcepool.memory.overhead.bytes`**
:   The amount of memory resource (in bytes) that will be used by a virtual machine above its guest memory requirements.

type: long

format: bytes


**`vsphere.resourcepool.memory.overhead.consumed.bytes`**
:   The amount of overhead memory, in bytes, currently being consumed to run a VM.

type: long

format: bytes


**`vsphere.resourcepool.memory.compressed.bytes`**
:   The amount of compressed memory currently consumed by VM, in bytes.

type: long

format: bytes


**`vsphere.resourcepool.name`**
:   The name of the resource pool.

type: keyword


**`vsphere.resourcepool.status`**
:   The overall health status of a host in the vSphere environment.

type: keyword


**`vsphere.resourcepool.vm.count`**
:   Number of virtual machines on the resource pool.

type: long


**`vsphere.resourcepool.vm.names`**
:   Names of virtual machines on the resource pool.

type: keyword


**`vsphere.resourcepool.triggered_alarms.*`**
:   List of all the triggered alarms.

type: object



## virtualmachine [_virtualmachine]

virtualmachine

**`vsphere.virtualmachine.host.id`**
:   Host id.

type: keyword


**`vsphere.virtualmachine.host.hostname`**
:   Hostname of the host.

type: keyword


**`vsphere.virtualmachine.id`**
:   Unique virtual machine ID.

type: keyword


**`vsphere.virtualmachine.name`**
:   Virtual machine name.

type: keyword


**`vsphere.virtualmachine.os`**
:   Virtual machine Operating System name.

type: keyword


**`vsphere.virtualmachine.cpu.used.mhz`**
:   Used CPU in Mhz.

type: long


**`vsphere.virtualmachine.cpu.total.mhz`**
:   Total Reserved CPU in Mhz.

type: long


**`vsphere.virtualmachine.cpu.free.mhz`**
:   Available CPU in Mhz.

type: long


**`vsphere.virtualmachine.memory.used.guest.bytes`**
:   Used memory of Guest in bytes.

type: long

format: bytes


**`vsphere.virtualmachine.memory.used.host.bytes`**
:   Used memory of Host in bytes.

type: long

format: bytes


**`vsphere.virtualmachine.memory.total.guest.bytes`**
:   Total memory of Guest in bytes.

type: long

format: bytes


**`vsphere.virtualmachine.memory.free.guest.bytes`**
:   Free memory of Guest in bytes.

type: long

format: bytes


**`vsphere.virtualmachine.custom_fields`**
:   Custom fields.

type: object


**`vsphere.virtualmachine.network_names`**
:   Network names.

type: keyword


**`vsphere.virtualmachine.datastore.names`**
:   Names of the datastore associated to this virtualmachine.

type: keyword


**`vsphere.virtualmachine.datastore.count`**
:   Number of datastores associated to this virtualmachine.

type: long


**`vsphere.virtualmachine.network.names`**
:   Names of the networks associated to this virtualmachine.

type: keyword


**`vsphere.virtualmachine.network.count`**
:   Number of networks associated to this virtualmachine.

type: long


**`vsphere.virtualmachine.status`**
:   Overall health and status of a virtual machine.

type: keyword


**`vsphere.virtualmachine.uptime`**
:   The uptime of the VM in seconds.

type: long


**`vsphere.virtualmachine.snapshot.info.*`**
:   Details of the snapshots of this virtualmachine.

type: object


**`vsphere.virtualmachine.snapshot.count`**
:   The number of snapshots of this virtualmachine.

type: long


**`vsphere.virtualmachine.triggered_alarms.*`**
:   List of all the triggered alarms.

type: object


