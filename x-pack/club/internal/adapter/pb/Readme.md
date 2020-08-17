# pb
--
    import "."

pb wraps packetbeat so the sniffer and analyzers can be used as inputs.

## Input "sniffer" settings

**interface.devices**: List of devices to collect packets from. An independent
sniffer and set of network analyzers will be run per device.

**interface.type**: Sniffer type. For example af_packet or pcap

**interface.buffer_size_mb**:

**interface.auto_promisc_mode** (NOT yet implemented): Put device into promisc
mode.

**interface.snaplen**:

**interface.with_vlans**:

**interface.bpf_filter**:

**flows.X**: See packetbeat flows settings.

**protocols.X**: See packetbeat protocol settings.

**ignore_outgoing**:

## Usage

#### func  Plugin

```go
func Plugin() v2.Plugin
```
Plugin provides a v2 input plugin implementation of packetbeat that allows
packetbeat functionality as an input.

The input name is "sniffer".

Each input instance will be independent and can read from multiple devices.
Multiple "sniffer" inputs can be run concurrently within a single process.
