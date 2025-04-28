---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/packetbeat-loopback-interface.html
---

# Packetbeat Can't capture traffic from Windows loopback interface [packetbeat-loopback-interface]

The Windows TCP/IP stack does not implement a network loopback interface, making it difficult for Windows packet capture drivers to capture traffic from the loopback device (127.0.0.1 traffic). To resolve this issue, install [Npcap](https://nmap.org/npcap/) in WinPcap API-compatible mode and select the option to support loopback traffic. When you restart Windows, Npcap creates an Npcap Loopback Adapter that you can select to capture loopback traffic.

For the list of devices shown here, you would configure Packetbeat to use device `4`:

```sh
PS C:\Program Files\Packetbeat .\packetbeat.exe -devices
0: \Device\NPF_NdisWanBh (NdisWan Adapter)
1: \Device\NPF_NdisWanIp (NdisWan Adapter)
2: \Device\NPF_NdisWanIpv6 (NdisWan Adapter)
3: \Device\NPF_{DD72B02C-4E48-4924-8D0F-F80EA2755534} (Intel(R) PRO/1000 MT Desktop Adapter)
4: \Device\NPF_{77DFFCAF-1335-4B0D-AFD4-5A4685674FAA} (MS NDIS 6.0 LoopBack Driver)
```

