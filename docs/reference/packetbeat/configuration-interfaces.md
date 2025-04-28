---
navigation_title: "Traffic sniffing"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/configuration-interfaces.html
---

# Configure traffic capturing options [configuration-interfaces]


There are two main ways of deploying Packetbeat:

* On dedicated servers, getting the traffic from mirror ports or tap devices.
* On your existing application servers.

The first option has the big advantage that there is no overhead of any kind on your application servers. But it requires dedicated networking gear, which is generally not available on cloud setups.

In both cases, the sniffing performance (reading packets passively from the network) is very important. In the case of a dedicated server, better sniffing performance means that less hardware is required. When Packetbeat is installed on an existing application server, better sniffing performance means less overhead.

Currently Packetbeat has several options for traffic capturing:

* `pcap`, which uses the libpcap library and works on most platforms, but it’s not the fastest option.
* `af_packet`, which uses memory mapped sniffing. This option is faster than libpcap and doesn’t require a kernel module, but it’s Linux-specific.

The `af_packet` option, also known as "memory-mapped sniffing," makes use of a Linux-specific [feature](https://www.kernel.org/doc/Documentation/networking/packet_mmap.txt). This could be the optimal sniffing mode for both the dedicated server and when Packetbeat is deployed on an existing application server.

The way it works is that both the kernel and the user space program map the same memory zone, and a simple circular buffer is organized in this memory zone. The kernel writes packets into the circular buffer, and the user space program reads from it. The poll system call is used for getting a notification for the first packet available, but the remaining available packets can be simply read via memory access.

The `af_packet` sniffer can be further tuned to use more memory in exchange for better performance. The larger the size of the circular buffer, the fewer system calls are needed, which means that fewer CPU cycles are consumed. The default size of the buffer is 30 MB, but you can increase it like this:

```yaml
packetbeat.interfaces.device: eth0
packetbeat.interfaces.type: af_packet
packetbeat.interfaces.buffer_size_mb: 100
```


## Windows Npcap installation options [_windows_npcap_installation_options]

On Windows Packetbeat requires an Npcap DLL installation. This is provided by Packetbeat for users of the Elastic Licenced version. In some cases users may wish to use their own installed version. In order to do this the `packetbeat.npcap.never_install` option can be used. Setting this option to `true` will not attempt to install the bundled Npcap library on start-up unless no Npcap is already installed.

```yaml
packetbeat.npcap.never_install: true
```


## Sniffing configuration options [_sniffing_configuration_options]

You can specify the following options in the `packetbeat.interfaces` section of the `packetbeat.yml` config file. Here is an example configuration:

```yaml
packetbeat.interfaces.device: any
packetbeat.interfaces.snaplen: 1514
packetbeat.interfaces.type: af_packet
packetbeat.interfaces.buffer_size_mb: 100
```


### `device` [_device]

The network device to capture traffic from. The specified device is set automatically to promiscuous mode, meaning that Packetbeat can capture traffic from other hosts on the same LAN.

Example:

```yaml
packetbeat.interfaces.device: eth0
```

On Linux, you can specify `any` for the device, and Packetbeat captures all messages sent or received by the server where Packetbeat is installed.

::::{note}
When you specify `any` for the device, the interfaces are not set to promiscuous mode.
::::


The `device` option also accepts specifying the device by its index in the list of devices available for sniffing. To obtain the list of available devices, run Packetbeat with the following command:

```sh
packetbeat devices
```

This command returns a list that looks something like the following:

```sh
0: en0 (No description available)
1: awdl0 (No description available)
2: bridge0 (No description available)
3: fw0 (No description available)
4: en1 (No description available)
5: en2 (No description available)
6: p2p0 (No description available)
7: en4 (No description available)
8: lo0 (No description available)
```

The following example sets up sniffing on the first interface in the list:

```yaml
packetbeat.interfaces.device: 0
```

Specifying the index is especially useful on Windows where device names can be long.

Alternatively the `default_route`, `default_route_ipv4` or `default_route_ipv6` device may be specified. This will set the capture device to be the device that is associated with the first default route identified on Packetbeat start-up. `default_route` will select the first default route from either IPv4 or IPv6 with a preference for the IPv4 route, while `default_route_ipv4` and `default_route_ipv6` will only select from the specified stack. The selected interface is not altered after it is chosen.


### `snaplen` [_snaplen]

The maximum size of the packets to capture. The default is 65535, which is large enough for almost all networks and interface types. If you sniff on a physical network interface, the optimal setting is the MTU size. On virtual interfaces, however, it’s safer to accept the default value.

Example:

```yaml
packetbeat.interfaces.device: eth0
packetbeat.interfaces.snaplen: 1514
```


### `type` [_type]

Packetbeat supports these sniffer types:

* `pcap`, which uses the libpcap library and works on most platforms, but it’s not the fastest option.
* `af_packet`, which uses memory-mapped sniffing. This option is faster than libpcap and doesn’t require a kernel module, but it’s Linux-specific.

The default sniffer type is `pcap`.

Here is an example configuration that specifies the `af_packet` sniffing type:

```yaml
packetbeat.interfaces.device: eth0
packetbeat.interfaces.type: af_packet
```

On Linux, if you are trying to optimize the CPU usage of Packetbeat, we recommend trying the `af_packet` option.

If you use the `af_packet` sniffer, you can tune its behaviour by specifying the following options:


### `buffer_size_mb` [_buffer_size_mb]

The maximum size of the shared memory buffer to use between the kernel and user space. A bigger buffer usually results in lower CPU usage, but consumes more memory. This setting is only available for the `af_packet` sniffer type. The default is 30 MB.

Example:

```yaml
packetbeat.interfaces.device: eth0
packetbeat.interfaces.type: af_packet
packetbeat.interfaces.buffer_size_mb: 100
```


### `fanout_group` [_fanout_group]

To scale processing across multiple Packetbeat processes, a fanout group identifier can be specified. When `fanout_group` is used the Linux kernel splits packets across Packetbeat instances in the same group by using a flow hash. It computes the flow hash modulo with the number of Packetbeat processes in order to consistently route flows to the same Packetbeat instance.

The value must be between 0 and 65535. By default, no value is set.

This is only available on Linux and requires using `type: af_packet`. Each process must be running in same network namespace. All processes must use the same interface settings. You must take responsibility for running multiple instances of Packetbeat.

Example:

```yaml
packetbeat.interfaces.type: af_packet
packetbeat.interfaces.fanout_group: 1
```


### `metrics_interval` [_metrics_interval]

Configure the metrics polling interval for supported interface types. Currently, only `af_packet` is supported.

The value must be a duration string. The default is `5s` (5 seconds). A value less than or equal to zero will be set to the default value.

Example:

```yaml
packetbeat.interfaces.type: af_packet
packetbeat.interfaces.metrics_interval: 5s
```


### `auto_promisc_mode` [_auto_promisc_mode]

With `auto_promisc_mode` Packetbeat puts interface in promiscuous mode automatically on startup. This option does not work with `any` interface device. The default option is false and requires manual set-up of promiscuous mode. Warning: under some circumstances (e.g beat crash) promiscuous mode can stay enabled even after beat is shut down.

Example:

```yaml
packetbeat.interfaces.device: eth0
packetbeat.interfaces.type: af_packet
packetbeat.interfaces.buffer_size_mb: 100
packetbeat.interfaces.auto_promisc_mode: true
```


### `with_vlans` [_with_vlans]

Packetbeat automatically generates a [BPF](https://en.wikipedia.org/wiki/Berkeley_Packet_Filter) for capturing only the traffic on ports where it expects to find known protocols. For example, if you have configured port 80 for HTTP and port 3306 for MySQL, Packetbeat generates the following BPF filter: `"port 80 or port 3306"`.

However, if the traffic contains [VLAN](https://en.wikipedia.org/wiki/IEEE_802.1Q) tags, the filter that Packetbeat generates is ineffective because the offset is moved by four bytes. To fix this, you can enable the `with_vlans` option, which generates a BPF filter that looks like this: `"port 80 or port 3306 or (vlan and (port 80 or port 3306))"`.


### `bpf_filter` [_bpf_filter]

Packetbeat automatically generates a [BPF](https://en.wikipedia.org/wiki/Berkeley_Packet_Filter) for capturing only the traffic on ports where it expects to find known protocols. For example, if you have configured port 80 for HTTP and port 3306 for MySQL, Packetbeat generates the following BPF filter: `"port 80 or port 3306"`.

You can use the `bpf_filter` setting to overwrite the generated BPF filter. For example:

```yaml
packetbeat.interfaces.device: eth0
packetbeat.interfaces.bpf_filter: "net 192.168.238.0/0 and port 80 or port 3306"
```

::::{note}
This setting disables automatic generation of the BPF filter. If you use this setting, it’s your responsibility to keep the BPF filters in sync with the ports defined in the `protocols` section.
::::



### `ignore_outgoing` [_ignore_outgoing]

If the `ignore_outgoing` option is enabled, Packetbeat ignores all the transactions initiated from the server running Packetbeat.

This is useful when two Packetbeat instances publish the same transactions. Because one Packetbeat sees the transaction in its outgoing queue and the other sees it in its incoming queue, you can end up with duplicate transactions. To remove the duplicates, you can enable the `packetbeat.ignore_outgoing` option on one of the servers.

For example, in the following scenario, you see a 3-server architecture where a Beat is installed on each server. t1 is the transaction exchanged between Server1 and Server2, and t2 is the transaction between Server2 and Server3.

![Beats Architecture](images/option_ignore_outgoing.png)

By default, each transaction is indexed twice because Beat2 sees both transactions. So you would see the following published transactions (when `ignore_outgoing` is false):

* Beat1: t1
* Beat2: t1 and t2
* Beat3: t2

To avoid duplicates, you can force your Beats to send only the incoming transactions and ignore the transactions created by the local server. So you would see the following published transactions (when `ignore_outgoing` is true):

* Beat1: none
* Beat2: t1
* Beat3: t2


### `internal_networks` [_internal_networks]

If the `internal_networks` option is specified, when monitoring network taps or mirror ports, Packetbeat will attempt to classify the network directionality of traffic not intended for this host as it relates to a network perimeter. Any CIDR block specified in `internal_networks` is treated internal to the perimeter, and any IP address falling outside of these CIDR blocks is considered external.

This is useful when Packetbeat is running on an appliance that sits at a network boundary such as a firewall or VPN. Note that this only affects how the directionality of network traffic is classified.

