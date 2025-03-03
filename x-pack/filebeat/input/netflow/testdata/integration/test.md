[integration/test.pcap](test.pcap) is a packet capture file that contains all the necessary Flow templates and records
in the correct sequence so that either with a single worker or multi workers (enabled template LRU) netflow input
produces the same number of `32` events. A snapshot of those extracted with netflow v2.18.0 integration installed is
shown below. The reason for relying only on checking the number of events in the integration test is to reduce the test
flaky-ness by making it less error-prone to future integration field changes.

```json
{
  "took": 3,
  "timed_out": false,
  "_shards": {
    "total": 1,
    "successful": 1,
    "skipped": 0,
    "failed": 0
  },
  "hits": {
    "total": {
      "value": 32,
      "relation": "eq"
    },
    "max_score": 1.0,
    "hits": [
      {
        "_index": ".ds-logs-netflow.log-default-2024.07.28-000001",
        "_id": "YvpB-5ABDjNvB3XCq_Q2",
        "_score": 1.0,
        "_source": {
          "agent": {
            "name": "lima-linux",
            "id": "6b1366c7-923c-4385-b168-7731c563d78e",
            "ephemeral_id": "e9cd68c6-0c95-40f5-a20f-7af8dfe49ffe",
            "type": "filebeat",
            "version": "8.16.0"
          },
          "destination": {
            "port": 61137,
            "ip": "10.100.11.14",
            "locality": "internal",
            "mac": "B4-FB-E4-D0-EA-7B"
          },
          "source": {
            "port": 993,
            "bytes": 298,
            "ip": "17.42.251.56",
            "locality": "external",
            "mac": "56-E0-32-C1-82-07",
            "packets": 4
          },
          "network": {
            "community_id": "1:/kw4vWmSSwJD+zp6pUP//kxYUR8=",
            "bytes": 298,
            "transport": "tcp",
            "type": "ipv4",
            "iana_number": "6",
            "packets": 4,
            "direction": "inbound"
          },
          "observer": {
            "ip": [
              "127.0.0.1"
            ]
          },
          "input": {
            "type": "netflow"
          },
          "netflow": {
            "packet_delta_count": 4,
            "protocol_identifier": 6,
            "vlan_id": 0,
            "source_mac_address": "56-E0-32-C1-82-07",
            "flow_start_sys_up_time": 2383112,
            "egress_interface": 4,
            "octet_delta_count": 298,
            "type": "netflow_flow",
            "destination_ipv4_address": "10.100.11.14",
            "source_ipv4_address": "17.42.251.56",
            "delta_flow_count": 0,
            "exporter": {
              "uptime_millis": 2446901,
              "address": "127.0.0.1:45715",
              "source_id": 0,
              "version": 9,
              "timestamp": "2024-06-13T23:29:24.000Z"
            },
            "tcp_control_bits": 24,
            "ip_version": 4,
            "ip_class_of_service": 0,
            "flow_direction": 0,
            "mpls_label_stack_length": 3,
            "ingress_interface": 3,
            "destination_mac_address": "B4-FB-E4-D0-EA-7B",
            "flow_end_sys_up_time": 2383171,
            "source_transport_port": 993,
            "destination_transport_port": 61137
          },
          "@timestamp": "2024-06-13T23:29:24.000Z",
          "related": {
            "ip": [
              "10.100.11.14",
              "17.42.251.56"
            ]
          },
          "ecs": {
            "version": "8.11.0"
          },
          "data_stream": {
            "namespace": "default",
            "type": "logs",
            "dataset": "netflow.log"
          },
          "host": {
            "hostname": "lima-linux",
            "os": {
              "kernel": "6.8.0-39-generic",
              "codename": "noble",
              "name": "Ubuntu",
              "type": "linux",
              "family": "debian",
              "version": "24.04 LTS (Noble Numbat)",
              "platform": "ubuntu"
            },
            "containerized": false,
            "ip": [
              "192.168.5.15",
              "fe80::5055:55ff:fed3:c3fa",
              "192.168.64.11",
              "fd7a:388c:fc26:8787:5055:55ff:fe57:4f30",
              "fe80::5055:55ff:fe57:4f30",
              "172.17.0.1",
              "172.18.0.1",
              "fe80::42:6eff:fef8:156a",
              "172.19.0.1",
              "fe80::42:23ff:fe61:a7e5",
              "fe80::dc46:b8ff:feeb:7268",
              "fe80::70be:2dff:fef5:9648",
              "fe80::dc52:efff:feac:491c"
            ],
            "name": "lima-linux",
            "id": "2efcdad36f7542db9d7212e71eafeebe",
            "mac": [
              "02-42-23-61-A7-E5",
              "02-42-33-CC-48-A9",
              "02-42-6E-F8-15-6A",
              "52-55-55-57-4F-30",
              "52-55-55-D3-C3-FA",
              "72-BE-2D-F5-96-48",
              "DE-46-B8-EB-72-68",
              "DE-52-EF-AC-49-1C"
            ],
            "architecture": "aarch64"
          },
          "event": {
            "duration": 59000000,
            "agent_id_status": "auth_metadata_missing",
            "ingested": "2024-07-28T21:31:43Z",
            "kind": "event",
            "created": "2024-07-28T21:31:36.104Z",
            "start": "2024-06-13T23:28:20.211Z",
            "action": "netflow_flow",
            "end": "2024-06-13T23:28:20.270Z",
            "category": [
              "network"
            ],
            "type": [
              "connection"
            ],
            "dataset": "netflow.log"
          },
          "flow": {
            "locality": "external",
            "id": "kOQywjrRMT4"
          }
        }
      },
      {
        "_index": ".ds-logs-netflow.log-default-2024.07.28-000001",
        "_id": "Y_pB-5ABDjNvB3XCq_Q2",
        "_score": 1.0,
        "_source": {
          "agent": {
            "name": "lima-linux",
            "id": "6b1366c7-923c-4385-b168-7731c563d78e",
            "ephemeral_id": "e9cd68c6-0c95-40f5-a20f-7af8dfe49ffe",
            "type": "filebeat",
            "version": "8.16.0"
          },
          "destination": {
            "port": 65058,
            "ip": "10.100.11.10",
            "locality": "internal",
            "mac": "B4-FB-E4-D0-EA-7B"
          },
          "source": {
            "port": 5223,
            "bytes": 846,
            "ip": "17.57.147.5",
            "locality": "external",
            "mac": "56-E0-32-C1-82-07",
            "packets": 2
          },
          "network": {
            "community_id": "1:JVogtRH3XanHiXtN+KyNkFU75VI=",
            "bytes": 846,
            "transport": "tcp",
            "type": "ipv4",
            "iana_number": "6",
            "packets": 2,
            "direction": "inbound"
          },
          "observer": {
            "ip": [
              "127.0.0.1"
            ]
          },
          "input": {
            "type": "netflow"
          },
          "netflow": {
            "protocol_identifier": 6,
            "packet_delta_count": 2,
            "vlan_id": 0,
            "flow_start_sys_up_time": 2383988,
            "source_mac_address": "56-E0-32-C1-82-07",
            "octet_delta_count": 846,
            "egress_interface": 4,
            "type": "netflow_flow",
            "destination_ipv4_address": "10.100.11.10",
            "source_ipv4_address": "17.57.147.5",
            "delta_flow_count": 0,
            "exporter": {
              "uptime_millis": 2446901,
              "address": "127.0.0.1:45715",
              "source_id": 0,
              "version": 9,
              "timestamp": "2024-06-13T23:29:24.000Z"
            },
            "tcp_control_bits": 24,
            "ip_class_of_service": 0,
            "ip_version": 4,
            "flow_direction": 0,
            "mpls_label_stack_length": 3,
            "ingress_interface": 3,
            "destination_mac_address": "B4-FB-E4-D0-EA-7B",
            "flow_end_sys_up_time": 2384006,
            "source_transport_port": 5223,
            "destination_transport_port": 65058
          },
          "@timestamp": "2024-06-13T23:29:24.000Z",
          "ecs": {
            "version": "8.11.0"
          },
          "related": {
            "ip": [
              "10.100.11.10",
              "17.57.147.5"
            ]
          },
          "data_stream": {
            "namespace": "default",
            "type": "logs",
            "dataset": "netflow.log"
          },
          "host": {
            "hostname": "lima-linux",
            "os": {
              "kernel": "6.8.0-39-generic",
              "codename": "noble",
              "name": "Ubuntu",
              "type": "linux",
              "family": "debian",
              "version": "24.04 LTS (Noble Numbat)",
              "platform": "ubuntu"
            },
            "containerized": false,
            "ip": [
              "192.168.5.15",
              "fe80::5055:55ff:fed3:c3fa",
              "192.168.64.11",
              "fd7a:388c:fc26:8787:5055:55ff:fe57:4f30",
              "fe80::5055:55ff:fe57:4f30",
              "172.17.0.1",
              "172.18.0.1",
              "fe80::42:6eff:fef8:156a",
              "172.19.0.1",
              "fe80::42:23ff:fe61:a7e5",
              "fe80::dc46:b8ff:feeb:7268",
              "fe80::70be:2dff:fef5:9648",
              "fe80::dc52:efff:feac:491c"
            ],
            "name": "lima-linux",
            "id": "2efcdad36f7542db9d7212e71eafeebe",
            "mac": [
              "02-42-23-61-A7-E5",
              "02-42-33-CC-48-A9",
              "02-42-6E-F8-15-6A",
              "52-55-55-57-4F-30",
              "52-55-55-D3-C3-FA",
              "72-BE-2D-F5-96-48",
              "DE-46-B8-EB-72-68",
              "DE-52-EF-AC-49-1C"
            ],
            "architecture": "aarch64"
          },
          "event": {
            "duration": 18000000,
            "agent_id_status": "auth_metadata_missing",
            "ingested": "2024-07-28T21:31:43Z",
            "created": "2024-07-28T21:31:36.104Z",
            "kind": "event",
            "start": "2024-06-13T23:28:21.087Z",
            "action": "netflow_flow",
            "end": "2024-06-13T23:28:21.105Z",
            "type": [
              "connection"
            ],
            "category": [
              "network"
            ],
            "dataset": "netflow.log"
          },
          "flow": {
            "locality": "external",
            "id": "EXqS-Ey8D6o"
          }
        }
      },
      {
        "_index": ".ds-logs-netflow.log-default-2024.07.28-000001",
        "_id": "ZPpB-5ABDjNvB3XCq_Q2",
        "_score": 1.0,
        "_source": {
          "agent": {
            "name": "lima-linux",
            "id": "6b1366c7-923c-4385-b168-7731c563d78e",
            "ephemeral_id": "e9cd68c6-0c95-40f5-a20f-7af8dfe49ffe",
            "type": "filebeat",
            "version": "8.16.0"
          },
          "destination": {
            "port": 45884,
            "ip": "10.100.8.34",
            "locality": "internal",
            "mac": "B4-FB-E4-D0-EA-7B"
          },
          "source": {
            "port": 853,
            "bytes": 3273,
            "ip": "1.0.0.1",
            "locality": "external",
            "packets": 16,
            "mac": "56-E0-32-C1-82-07"
          },
          "network": {
            "community_id": "1:7zyW7exctP8D685WBGJPtxtT1xs=",
            "bytes": 3273,
            "transport": "tcp",
            "type": "ipv4",
            "iana_number": "6",
            "packets": 16,
            "direction": "inbound"
          },
          "input": {
            "type": "netflow"
          },
          "observer": {
            "ip": [
              "127.0.0.1"
            ]
          },
          "netflow": {
            "packet_delta_count": 16,
            "protocol_identifier": 6,
            "vlan_id": 0,
            "flow_start_sys_up_time": 2381818,
            "source_mac_address": "56-E0-32-C1-82-07",
            "egress_interface": 4,
            "octet_delta_count": 3273,
            "type": "netflow_flow",
            "destination_ipv4_address": "10.100.8.34",
            "source_ipv4_address": "1.0.0.1",
            "exporter": {
              "uptime_millis": 2446901,
              "address": "127.0.0.1:45715",
              "source_id": 0,
              "version": 9,
              "timestamp": "2024-06-13T23:29:24.000Z"
            },
            "delta_flow_count": 0,
            "tcp_control_bits": 30,
            "ip_version": 4,
            "ip_class_of_service": 0,
            "flow_direction": 0,
            "mpls_label_stack_length": 3,
            "ingress_interface": 3,
            "destination_mac_address": "B4-FB-E4-D0-EA-7B",
            "flow_end_sys_up_time": 2434554,
            "source_transport_port": 853,
            "destination_transport_port": 45884
          },
          "@timestamp": "2024-06-13T23:29:24.000Z",
          "related": {
            "ip": [
              "1.0.0.1",
              "10.100.8.34"
            ]
          },
          "ecs": {
            "version": "8.11.0"
          },
          "data_stream": {
            "namespace": "default",
            "type": "logs",
            "dataset": "netflow.log"
          },
          "host": {
            "hostname": "lima-linux",
            "os": {
              "kernel": "6.8.0-39-generic",
              "codename": "noble",
              "name": "Ubuntu",
              "type": "linux",
              "family": "debian",
              "version": "24.04 LTS (Noble Numbat)",
              "platform": "ubuntu"
            },
            "containerized": false,
            "ip": [
              "192.168.5.15",
              "fe80::5055:55ff:fed3:c3fa",
              "192.168.64.11",
              "fd7a:388c:fc26:8787:5055:55ff:fe57:4f30",
              "fe80::5055:55ff:fe57:4f30",
              "172.17.0.1",
              "172.18.0.1",
              "fe80::42:6eff:fef8:156a",
              "172.19.0.1",
              "fe80::42:23ff:fe61:a7e5",
              "fe80::dc46:b8ff:feeb:7268",
              "fe80::70be:2dff:fef5:9648",
              "fe80::dc52:efff:feac:491c"
            ],
            "name": "lima-linux",
            "id": "2efcdad36f7542db9d7212e71eafeebe",
            "mac": [
              "02-42-23-61-A7-E5",
              "02-42-33-CC-48-A9",
              "02-42-6E-F8-15-6A",
              "52-55-55-57-4F-30",
              "52-55-55-D3-C3-FA",
              "72-BE-2D-F5-96-48",
              "DE-46-B8-EB-72-68",
              "DE-52-EF-AC-49-1C"
            ],
            "architecture": "aarch64"
          },
          "event": {
            "duration": 52736000000,
            "agent_id_status": "auth_metadata_missing",
            "ingested": "2024-07-28T21:31:43Z",
            "created": "2024-07-28T21:31:38.073Z",
            "kind": "event",
            "start": "2024-06-13T23:28:18.917Z",
            "action": "netflow_flow",
            "end": "2024-06-13T23:29:11.653Z",
            "category": [
              "network"
            ],
            "type": [
              "connection"
            ],
            "dataset": "netflow.log"
          },
          "flow": {
            "locality": "external",
            "id": "1F9ai5fhOyY"
          }
        }
      },
      {
        "_index": ".ds-logs-netflow.log-default-2024.07.28-000001",
        "_id": "ZfpB-5ABDjNvB3XCq_Q2",
        "_score": 1.0,
        "_source": {
          "agent": {
            "name": "lima-linux",
            "id": "6b1366c7-923c-4385-b168-7731c563d78e",
            "ephemeral_id": "e9cd68c6-0c95-40f5-a20f-7af8dfe49ffe",
            "type": "filebeat",
            "version": "8.16.0"
          },
          "destination": {
            "port": 56290,
            "ip": "10.100.8.34",
            "locality": "internal",
            "mac": "B4-FB-E4-D0-EA-7B"
          },
          "source": {
            "port": 853,
            "bytes": 3402,
            "ip": "1.0.0.1",
            "locality": "external",
            "mac": "56-E0-32-C1-82-07",
            "packets": 15
          },
          "network": {
            "community_id": "1:JiM5h5WwM8mL6KbDc2FcKTrW6l0=",
            "bytes": 3402,
            "transport": "tcp",
            "type": "ipv4",
            "iana_number": "6",
            "packets": 15,
            "direction": "inbound"
          },
          "input": {
            "type": "netflow"
          },
          "observer": {
            "ip": [
              "127.0.0.1"
            ]
          },
          "netflow": {
            "protocol_identifier": 6,
            "packet_delta_count": 15,
            "vlan_id": 0,
            "source_mac_address": "56-E0-32-C1-82-07",
            "flow_start_sys_up_time": 2402156,
            "octet_delta_count": 3402,
            "egress_interface": 4,
            "type": "netflow_flow",
            "destination_ipv4_address": "10.100.8.34",
            "source_ipv4_address": "1.0.0.1",
            "exporter": {
              "uptime_millis": 2446901,
              "address": "127.0.0.1:45715",
              "source_id": 0,
              "version": 9,
              "timestamp": "2024-06-13T23:29:24.000Z"
            },
            "delta_flow_count": 0,
            "tcp_control_bits": 30,
            "ip_version": 4,
            "ip_class_of_service": 0,
            "flow_direction": 0,
            "mpls_label_stack_length": 3,
            "ingress_interface": 3,
            "destination_mac_address": "B4-FB-E4-D0-EA-7B",
            "flow_end_sys_up_time": 2434554,
            "source_transport_port": 853,
            "destination_transport_port": 56290
          },
          "@timestamp": "2024-06-13T23:29:24.000Z",
          "related": {
            "ip": [
              "1.0.0.1",
              "10.100.8.34"
            ]
          },
          "ecs": {
            "version": "8.11.0"
          },
          "data_stream": {
            "namespace": "default",
            "type": "logs",
            "dataset": "netflow.log"
          },
          "host": {
            "hostname": "lima-linux",
            "os": {
              "kernel": "6.8.0-39-generic",
              "codename": "noble",
              "name": "Ubuntu",
              "family": "debian",
              "type": "linux",
              "version": "24.04 LTS (Noble Numbat)",
              "platform": "ubuntu"
            },
            "containerized": false,
            "ip": [
              "192.168.5.15",
              "fe80::5055:55ff:fed3:c3fa",
              "192.168.64.11",
              "fd7a:388c:fc26:8787:5055:55ff:fe57:4f30",
              "fe80::5055:55ff:fe57:4f30",
              "172.17.0.1",
              "172.18.0.1",
              "fe80::42:6eff:fef8:156a",
              "172.19.0.1",
              "fe80::42:23ff:fe61:a7e5",
              "fe80::dc46:b8ff:feeb:7268",
              "fe80::70be:2dff:fef5:9648",
              "fe80::dc52:efff:feac:491c"
            ],
            "name": "lima-linux",
            "id": "2efcdad36f7542db9d7212e71eafeebe",
            "mac": [
              "02-42-23-61-A7-E5",
              "02-42-33-CC-48-A9",
              "02-42-6E-F8-15-6A",
              "52-55-55-57-4F-30",
              "52-55-55-D3-C3-FA",
              "72-BE-2D-F5-96-48",
              "DE-46-B8-EB-72-68",
              "DE-52-EF-AC-49-1C"
            ],
            "architecture": "aarch64"
          },
          "event": {
            "duration": 32398000000,
            "agent_id_status": "auth_metadata_missing",
            "ingested": "2024-07-28T21:31:43Z",
            "created": "2024-07-28T21:31:38.073Z",
            "kind": "event",
            "start": "2024-06-13T23:28:39.255Z",
            "action": "netflow_flow",
            "end": "2024-06-13T23:29:11.653Z",
            "type": [
              "connection"
            ],
            "category": [
              "network"
            ],
            "dataset": "netflow.log"
          },
          "flow": {
            "locality": "external",
            "id": "t2rofv-PTS8"
          }
        }
      },
      {
        "_index": ".ds-logs-netflow.log-default-2024.07.28-000001",
        "_id": "ZvpB-5ABDjNvB3XCq_Q2",
        "_score": 1.0,
        "_source": {
          "agent": {
            "name": "lima-linux",
            "id": "6b1366c7-923c-4385-b168-7731c563d78e",
            "type": "filebeat",
            "ephemeral_id": "e9cd68c6-0c95-40f5-a20f-7af8dfe49ffe",
            "version": "8.16.0"
          },
          "destination": {
            "port": 48478,
            "ip": "10.100.8.38",
            "locality": "internal",
            "mac": "B4-FB-E4-D0-EA-7B"
          },
          "source": {
            "port": 443,
            "bytes": 7177,
            "ip": "54.160.25.132",
            "locality": "external",
            "mac": "56-E0-32-C1-82-07",
            "packets": 13
          },
          "network": {
            "community_id": "1:dPba87I3GhRefpqzpLtFV1FmSCc=",
            "bytes": 7177,
            "transport": "tcp",
            "type": "ipv4",
            "iana_number": "6",
            "packets": 13,
            "direction": "inbound"
          },
          "input": {
            "type": "netflow"
          },
          "observer": {
            "ip": [
              "127.0.0.1"
            ]
          },
          "netflow": {
            "packet_delta_count": 13,
            "protocol_identifier": 6,
            "vlan_id": 0,
            "source_mac_address": "56-E0-32-C1-82-07",
            "flow_start_sys_up_time": 2382399,
            "octet_delta_count": 7177,
            "egress_interface": 4,
            "type": "netflow_flow",
            "destination_ipv4_address": "10.100.8.38",
            "source_ipv4_address": "54.160.25.132",
            "delta_flow_count": 0,
            "exporter": {
              "uptime_millis": 2446901,
              "address": "127.0.0.1:45715",
              "source_id": 0,
              "version": 9,
              "timestamp": "2024-06-13T23:29:24.000Z"
            },
            "tcp_control_bits": 27,
            "ip_version": 4,
            "ip_class_of_service": 0,
            "flow_direction": 0,
            "mpls_label_stack_length": 3,
            "ingress_interface": 3,
            "destination_mac_address": "B4-FB-E4-D0-EA-7B",
            "flow_end_sys_up_time": 2385519,
            "destination_transport_port": 48478,
            "source_transport_port": 443
          },
          "@timestamp": "2024-06-13T23:29:24.000Z",
          "related": {
            "ip": [
              "10.100.8.38",
              "54.160.25.132"
            ]
          },
          "ecs": {
            "version": "8.11.0"
          },
          "data_stream": {
            "namespace": "default",
            "type": "logs",
            "dataset": "netflow.log"
          },
          "host": {
            "hostname": "lima-linux",
            "os": {
              "kernel": "6.8.0-39-generic",
              "codename": "noble",
              "name": "Ubuntu",
              "type": "linux",
              "family": "debian",
              "version": "24.04 LTS (Noble Numbat)",
              "platform": "ubuntu"
            },
            "containerized": false,
            "ip": [
              "192.168.5.15",
              "fe80::5055:55ff:fed3:c3fa",
              "192.168.64.11",
              "fd7a:388c:fc26:8787:5055:55ff:fe57:4f30",
              "fe80::5055:55ff:fe57:4f30",
              "172.17.0.1",
              "172.18.0.1",
              "fe80::42:6eff:fef8:156a",
              "172.19.0.1",
              "fe80::42:23ff:fe61:a7e5",
              "fe80::dc46:b8ff:feeb:7268",
              "fe80::70be:2dff:fef5:9648",
              "fe80::dc52:efff:feac:491c"
            ],
            "name": "lima-linux",
            "id": "2efcdad36f7542db9d7212e71eafeebe",
            "mac": [
              "02-42-23-61-A7-E5",
              "02-42-33-CC-48-A9",
              "02-42-6E-F8-15-6A",
              "52-55-55-57-4F-30",
              "52-55-55-D3-C3-FA",
              "72-BE-2D-F5-96-48",
              "DE-46-B8-EB-72-68",
              "DE-52-EF-AC-49-1C"
            ],
            "architecture": "aarch64"
          },
          "event": {
            "duration": 3120000000,
            "agent_id_status": "auth_metadata_missing",
            "ingested": "2024-07-28T21:31:43Z",
            "created": "2024-07-28T21:31:38.073Z",
            "kind": "event",
            "start": "2024-06-13T23:28:19.498Z",
            "action": "netflow_flow",
            "end": "2024-06-13T23:28:22.618Z",
            "type": [
              "connection"
            ],
            "category": [
              "network"
            ],
            "dataset": "netflow.log"
          },
          "flow": {
            "locality": "external",
            "id": "faXLmv8YuVA"
          }
        }
      },
      {
        "_index": ".ds-logs-netflow.log-default-2024.07.28-000001",
        "_id": "Z_pB-5ABDjNvB3XCq_Q2",
        "_score": 1.0,
        "_source": {
          "agent": {
            "name": "lima-linux",
            "id": "6b1366c7-923c-4385-b168-7731c563d78e",
            "type": "filebeat",
            "ephemeral_id": "e9cd68c6-0c95-40f5-a20f-7af8dfe49ffe",
            "version": "8.16.0"
          },
          "destination": {
            "port": 50020,
            "ip": "71.191.210.227",
            "locality": "external",
            "mac": "B4-FB-E4-D0-EA-7B"
          },
          "source": {
            "port": 50020,
            "bytes": 1775780,
            "ip": "64.225.12.142",
            "locality": "external",
            "packets": 5439,
            "mac": "56-E0-32-C1-82-07"
          },
          "network": {
            "community_id": "1:5adPyESITZct6QsulanflJ1zzGw=",
            "bytes": 1775780,
            "transport": "udp",
            "type": "ipv4",
            "iana_number": "17",
            "packets": 5439,
            "direction": "external"
          },
          "input": {
            "type": "netflow"
          },
          "observer": {
            "ip": [
              "127.0.0.1"
            ]
          },
          "netflow": {
            "packet_delta_count": 5439,
            "protocol_identifier": 17,
            "vlan_id": 0,
            "flow_start_sys_up_time": 2386212,
            "source_mac_address": "56-E0-32-C1-82-07",
            "egress_interface": 0,
            "octet_delta_count": 1775780,
            "type": "netflow_flow",
            "destination_ipv4_address": "71.191.210.227",
            "source_ipv4_address": "64.225.12.142",
            "delta_flow_count": 0,
            "exporter": {
              "uptime_millis": 2447278,
              "address": "127.0.0.1:45715",
              "source_id": 0,
              "version": 9,
              "timestamp": "2024-06-13T23:29:25.000Z"
            },
            "tcp_control_bits": 0,
            "ip_version": 4,
            "ip_class_of_service": 0,
            "flow_direction": 0,
            "mpls_label_stack_length": 3,
            "ingress_interface": 3,
            "destination_mac_address": "B4-FB-E4-D0-EA-7B",
            "flow_end_sys_up_time": 2447242,
            "source_transport_port": 50020,
            "destination_transport_port": 50020
          },
          "@timestamp": "2024-06-13T23:29:25.000Z",
          "related": {
            "ip": [
              "64.225.12.142",
              "71.191.210.227"
            ]
          },
          "ecs": {
            "version": "8.11.0"
          },
          "data_stream": {
            "namespace": "default",
            "type": "logs",
            "dataset": "netflow.log"
          },
          "host": {
            "hostname": "lima-linux",
            "os": {
              "kernel": "6.8.0-39-generic",
              "codename": "noble",
              "name": "Ubuntu",
              "type": "linux",
              "family": "debian",
              "version": "24.04 LTS (Noble Numbat)",
              "platform": "ubuntu"
            },
            "ip": [
              "192.168.5.15",
              "fe80::5055:55ff:fed3:c3fa",
              "192.168.64.11",
              "fd7a:388c:fc26:8787:5055:55ff:fe57:4f30",
              "fe80::5055:55ff:fe57:4f30",
              "172.17.0.1",
              "172.18.0.1",
              "fe80::42:6eff:fef8:156a",
              "172.19.0.1",
              "fe80::42:23ff:fe61:a7e5",
              "fe80::dc46:b8ff:feeb:7268",
              "fe80::70be:2dff:fef5:9648",
              "fe80::dc52:efff:feac:491c"
            ],
            "containerized": false,
            "name": "lima-linux",
            "id": "2efcdad36f7542db9d7212e71eafeebe",
            "mac": [
              "02-42-23-61-A7-E5",
              "02-42-33-CC-48-A9",
              "02-42-6E-F8-15-6A",
              "52-55-55-57-4F-30",
              "52-55-55-D3-C3-FA",
              "72-BE-2D-F5-96-48",
              "DE-46-B8-EB-72-68",
              "DE-52-EF-AC-49-1C"
            ],
            "architecture": "aarch64"
          },
          "event": {
            "duration": 61030000000,
            "agent_id_status": "auth_metadata_missing",
            "ingested": "2024-07-28T21:31:43Z",
            "created": "2024-07-28T21:31:38.073Z",
            "kind": "event",
            "start": "2024-06-13T23:28:23.934Z",
            "action": "netflow_flow",
            "end": "2024-06-13T23:29:24.964Z",
            "category": [
              "network"
            ],
            "type": [
              "connection"
            ],
            "dataset": "netflow.log"
          },
          "flow": {
            "locality": "external",
            "id": "Z43o3EB3dqc"
          }
        }
      },
      {
        "_index": ".ds-logs-netflow.log-default-2024.07.28-000001",
        "_id": "aPpB-5ABDjNvB3XCq_Q2",
        "_score": 1.0,
        "_source": {
          "agent": {
            "name": "lima-linux",
            "id": "6b1366c7-923c-4385-b168-7731c563d78e",
            "type": "filebeat",
            "ephemeral_id": "e9cd68c6-0c95-40f5-a20f-7af8dfe49ffe",
            "version": "8.16.0"
          },
          "destination": {
            "port": 49212,
            "ip": "10.100.11.10",
            "locality": "internal",
            "mac": "B4-FB-E4-D0-EA-7B"
          },
          "source": {
            "port": 9243,
            "bytes": 238429,
            "ip": "3.215.12.84",
            "locality": "external",
            "packets": 169,
            "mac": "56-E0-32-C1-82-07"
          },
          "network": {
            "community_id": "1:SboTt968e79D3MOYTPiGam4R7e4=",
            "bytes": 238429,
            "transport": "tcp",
            "type": "ipv4",
            "packets": 169,
            "iana_number": "6",
            "direction": "inbound"
          },
          "observer": {
            "ip": [
              "127.0.0.1"
            ]
          },
          "input": {
            "type": "netflow"
          },
          "netflow": {
            "packet_delta_count": 169,
            "protocol_identifier": 6,
            "vlan_id": 0,
            "source_mac_address": "56-E0-32-C1-82-07",
            "flow_start_sys_up_time": 2386462,
            "octet_delta_count": 238429,
            "egress_interface": 4,
            "type": "netflow_flow",
            "destination_ipv4_address": "10.100.11.10",
            "source_ipv4_address": "3.215.12.84",
            "exporter": {
              "uptime_millis": 2447278,
              "address": "127.0.0.1:45715",
              "source_id": 0,
              "version": 9,
              "timestamp": "2024-06-13T23:29:25.000Z"
            },
            "delta_flow_count": 0,
            "tcp_control_bits": 24,
            "ip_class_of_service": 0,
            "ip_version": 4,
            "flow_direction": 0,
            "mpls_label_stack_length": 3,
            "ingress_interface": 3,
            "destination_mac_address": "B4-FB-E4-D0-EA-7B",
            "flow_end_sys_up_time": 2447094,
            "source_transport_port": 9243,
            "destination_transport_port": 49212
          },
          "@timestamp": "2024-06-13T23:29:25.000Z",
          "related": {
            "ip": [
              "3.215.12.84",
              "10.100.11.10"
            ]
          },
          "ecs": {
            "version": "8.11.0"
          },
          "data_stream": {
            "namespace": "default",
            "type": "logs",
            "dataset": "netflow.log"
          },
          "host": {
            "hostname": "lima-linux",
            "os": {
              "kernel": "6.8.0-39-generic",
              "codename": "noble",
              "name": "Ubuntu",
              "type": "linux",
              "family": "debian",
              "version": "24.04 LTS (Noble Numbat)",
              "platform": "ubuntu"
            },
            "ip": [
              "192.168.5.15",
              "fe80::5055:55ff:fed3:c3fa",
              "192.168.64.11",
              "fd7a:388c:fc26:8787:5055:55ff:fe57:4f30",
              "fe80::5055:55ff:fe57:4f30",
              "172.17.0.1",
              "172.18.0.1",
              "fe80::42:6eff:fef8:156a",
              "172.19.0.1",
              "fe80::42:23ff:fe61:a7e5",
              "fe80::dc46:b8ff:feeb:7268",
              "fe80::70be:2dff:fef5:9648",
              "fe80::dc52:efff:feac:491c"
            ],
            "containerized": false,
            "name": "lima-linux",
            "id": "2efcdad36f7542db9d7212e71eafeebe",
            "mac": [
              "02-42-23-61-A7-E5",
              "02-42-33-CC-48-A9",
              "02-42-6E-F8-15-6A",
              "52-55-55-57-4F-30",
              "52-55-55-D3-C3-FA",
              "72-BE-2D-F5-96-48",
              "DE-46-B8-EB-72-68",
              "DE-52-EF-AC-49-1C"
            ],
            "architecture": "aarch64"
          },
          "event": {
            "duration": 60632000000,
            "agent_id_status": "auth_metadata_missing",
            "ingested": "2024-07-28T21:31:43Z",
            "kind": "event",
            "created": "2024-07-28T21:31:38.073Z",
            "start": "2024-06-13T23:28:24.184Z",
            "action": "netflow_flow",
            "end": "2024-06-13T23:29:24.816Z",
            "category": [
              "network"
            ],
            "type": [
              "connection"
            ],
            "dataset": "netflow.log"
          },
          "flow": {
            "locality": "external",
            "id": "5dOC8RdfsHE"
          }
        }
      },
      {
        "_index": ".ds-logs-netflow.log-default-2024.07.28-000001",
        "_id": "afpB-5ABDjNvB3XCq_Q2",
        "_score": 1.0,
        "_source": {
          "agent": {
            "name": "lima-linux",
            "id": "6b1366c7-923c-4385-b168-7731c563d78e",
            "ephemeral_id": "e9cd68c6-0c95-40f5-a20f-7af8dfe49ffe",
            "type": "filebeat",
            "version": "8.16.0"
          },
          "destination": {
            "port": 53284,
            "ip": "10.100.8.98",
            "locality": "internal",
            "mac": "B4-FB-E4-D0-EA-7B"
          },
          "source": {
            "port": 443,
            "bytes": 8301,
            "ip": "34.234.143.15",
            "locality": "external",
            "mac": "56-E0-32-C1-82-07",
            "packets": 18
          },
          "network": {
            "community_id": "1:sgZy+IyajSLfG50OHXQKQiTmQ7s=",
            "bytes": 8301,
            "transport": "tcp",
            "type": "ipv4",
            "packets": 18,
            "iana_number": "6",
            "direction": "inbound"
          },
          "observer": {
            "ip": [
              "127.0.0.1"
            ]
          },
          "input": {
            "type": "netflow"
          },
          "netflow": {
            "packet_delta_count": 18,
            "protocol_identifier": 6,
            "vlan_id": 0,
            "flow_start_sys_up_time": 2371639,
            "source_mac_address": "56-E0-32-C1-82-07",
            "egress_interface": 4,
            "octet_delta_count": 8301,
            "type": "netflow_flow",
            "destination_ipv4_address": "10.100.8.98",
            "source_ipv4_address": "34.234.143.15",
            "delta_flow_count": 0,
            "exporter": {
              "uptime_millis": 2447278,
              "address": "127.0.0.1:45715",
              "source_id": 0,
              "version": 9,
              "timestamp": "2024-06-13T23:29:25.000Z"
            },
            "tcp_control_bits": 27,
            "ip_version": 4,
            "ip_class_of_service": 0,
            "flow_direction": 0,
            "mpls_label_stack_length": 3,
            "ingress_interface": 3,
            "destination_mac_address": "B4-FB-E4-D0-EA-7B",
            "flow_end_sys_up_time": 2386709,
            "source_transport_port": 443,
            "destination_transport_port": 53284
          },
          "@timestamp": "2024-06-13T23:29:25.000Z",
          "related": {
            "ip": [
              "10.100.8.98",
              "34.234.143.15"
            ]
          },
          "ecs": {
            "version": "8.11.0"
          },
          "data_stream": {
            "namespace": "default",
            "type": "logs",
            "dataset": "netflow.log"
          },
          "host": {
            "hostname": "lima-linux",
            "os": {
              "kernel": "6.8.0-39-generic",
              "codename": "noble",
              "name": "Ubuntu",
              "type": "linux",
              "family": "debian",
              "version": "24.04 LTS (Noble Numbat)",
              "platform": "ubuntu"
            },
            "ip": [
              "192.168.5.15",
              "fe80::5055:55ff:fed3:c3fa",
              "192.168.64.11",
              "fd7a:388c:fc26:8787:5055:55ff:fe57:4f30",
              "fe80::5055:55ff:fe57:4f30",
              "172.17.0.1",
              "172.18.0.1",
              "fe80::42:6eff:fef8:156a",
              "172.19.0.1",
              "fe80::42:23ff:fe61:a7e5",
              "fe80::dc46:b8ff:feeb:7268",
              "fe80::70be:2dff:fef5:9648",
              "fe80::dc52:efff:feac:491c"
            ],
            "containerized": false,
            "name": "lima-linux",
            "id": "2efcdad36f7542db9d7212e71eafeebe",
            "mac": [
              "02-42-23-61-A7-E5",
              "02-42-33-CC-48-A9",
              "02-42-6E-F8-15-6A",
              "52-55-55-57-4F-30",
              "52-55-55-D3-C3-FA",
              "72-BE-2D-F5-96-48",
              "DE-46-B8-EB-72-68",
              "DE-52-EF-AC-49-1C"
            ],
            "architecture": "aarch64"
          },
          "event": {
            "duration": 15070000000,
            "agent_id_status": "auth_metadata_missing",
            "ingested": "2024-07-28T21:31:43Z",
            "created": "2024-07-28T21:31:38.073Z",
            "kind": "event",
            "start": "2024-06-13T23:28:09.361Z",
            "action": "netflow_flow",
            "end": "2024-06-13T23:28:24.431Z",
            "category": [
              "network"
            ],
            "type": [
              "connection"
            ],
            "dataset": "netflow.log"
          },
          "flow": {
            "locality": "external",
            "id": "TXluQ-JewIU"
          }
        }
      },
      {
        "_index": ".ds-logs-netflow.log-default-2024.07.28-000001",
        "_id": "avpB-5ABDjNvB3XCq_Q2",
        "_score": 1.0,
        "_source": {
          "agent": {
            "name": "lima-linux",
            "id": "6b1366c7-923c-4385-b168-7731c563d78e",
            "type": "filebeat",
            "ephemeral_id": "e9cd68c6-0c95-40f5-a20f-7af8dfe49ffe",
            "version": "8.16.0"
          },
          "destination": {
            "port": 59242,
            "ip": "10.100.8.36",
            "locality": "internal",
            "mac": "B4-FB-E4-D0-EA-7B"
          },
          "source": {
            "port": 443,
            "bytes": 7068,
            "ip": "54.80.119.44",
            "locality": "external",
            "mac": "56-E0-32-C1-82-07",
            "packets": 11
          },
          "network": {
            "community_id": "1:WORfT191rnMqMqwKSM+88k8ngso=",
            "bytes": 7068,
            "transport": "tcp",
            "type": "ipv4",
            "iana_number": "6",
            "packets": 11,
            "direction": "inbound"
          },
          "input": {
            "type": "netflow"
          },
          "observer": {
            "ip": [
              "127.0.0.1"
            ]
          },
          "netflow": {
            "packet_delta_count": 11,
            "protocol_identifier": 6,
            "vlan_id": 0,
            "source_mac_address": "56-E0-32-C1-82-07",
            "flow_start_sys_up_time": 2383262,
            "octet_delta_count": 7068,
            "egress_interface": 4,
            "type": "netflow_flow",
            "destination_ipv4_address": "10.100.8.36",
            "source_ipv4_address": "54.80.119.44",
            "exporter": {
              "uptime_millis": 2447278,
              "address": "127.0.0.1:45715",
              "source_id": 0,
              "version": 9,
              "timestamp": "2024-06-13T23:29:25.000Z"
            },
            "delta_flow_count": 0,
            "tcp_control_bits": 27,
            "ip_class_of_service": 0,
            "ip_version": 4,
            "flow_direction": 0,
            "mpls_label_stack_length": 3,
            "ingress_interface": 3,
            "destination_mac_address": "B4-FB-E4-D0-EA-7B",
            "flow_end_sys_up_time": 2386300,
            "destination_transport_port": 59242,
            "source_transport_port": 443
          },
          "@timestamp": "2024-06-13T23:29:25.000Z",
          "related": {
            "ip": [
              "10.100.8.36",
              "54.80.119.44"
            ]
          },
          "ecs": {
            "version": "8.11.0"
          },
          "data_stream": {
            "namespace": "default",
            "type": "logs",
            "dataset": "netflow.log"
          },
          "host": {
            "hostname": "lima-linux",
            "os": {
              "kernel": "6.8.0-39-generic",
              "codename": "noble",
              "name": "Ubuntu",
              "type": "linux",
              "family": "debian",
              "version": "24.04 LTS (Noble Numbat)",
              "platform": "ubuntu"
            },
            "containerized": false,
            "ip": [
              "192.168.5.15",
              "fe80::5055:55ff:fed3:c3fa",
              "192.168.64.11",
              "fd7a:388c:fc26:8787:5055:55ff:fe57:4f30",
              "fe80::5055:55ff:fe57:4f30",
              "172.17.0.1",
              "172.18.0.1",
              "fe80::42:6eff:fef8:156a",
              "172.19.0.1",
              "fe80::42:23ff:fe61:a7e5",
              "fe80::dc46:b8ff:feeb:7268",
              "fe80::70be:2dff:fef5:9648",
              "fe80::dc52:efff:feac:491c"
            ],
            "name": "lima-linux",
            "id": "2efcdad36f7542db9d7212e71eafeebe",
            "mac": [
              "02-42-23-61-A7-E5",
              "02-42-33-CC-48-A9",
              "02-42-6E-F8-15-6A",
              "52-55-55-57-4F-30",
              "52-55-55-D3-C3-FA",
              "72-BE-2D-F5-96-48",
              "DE-46-B8-EB-72-68",
              "DE-52-EF-AC-49-1C"
            ],
            "architecture": "aarch64"
          },
          "event": {
            "duration": 3038000000,
            "agent_id_status": "auth_metadata_missing",
            "ingested": "2024-07-28T21:31:43Z",
            "created": "2024-07-28T21:31:38.073Z",
            "kind": "event",
            "start": "2024-06-13T23:28:20.984Z",
            "action": "netflow_flow",
            "end": "2024-06-13T23:28:24.022Z",
            "type": [
              "connection"
            ],
            "category": [
              "network"
            ],
            "dataset": "netflow.log"
          },
          "flow": {
            "locality": "external",
            "id": "woOLyw0G8JI"
          }
        }
      },
      {
        "_index": ".ds-logs-netflow.log-default-2024.07.28-000001",
        "_id": "a_pB-5ABDjNvB3XCq_Q2",
        "_score": 1.0,
        "_source": {
          "agent": {
            "name": "lima-linux",
            "id": "6b1366c7-923c-4385-b168-7731c563d78e",
            "type": "filebeat",
            "ephemeral_id": "e9cd68c6-0c95-40f5-a20f-7af8dfe49ffe",
            "version": "8.16.0"
          },
          "destination": {
            "port": 64047,
            "ip": "10.100.11.10",
            "locality": "internal",
            "mac": "B4-FB-E4-D0-EA-7B"
          },
          "source": {
            "port": 443,
            "bytes": 312,
            "ip": "34.71.180.220",
            "locality": "external",
            "mac": "56-E0-32-C1-82-07",
            "packets": 6
          },
          "network": {
            "community_id": "1:dghJP1SzJKaNyBEJsW2sueeiDGw=",
            "bytes": 312,
            "transport": "tcp",
            "type": "ipv4",
            "iana_number": "6",
            "packets": 6,
            "direction": "inbound"
          },
          "input": {
            "type": "netflow"
          },
          "observer": {
            "ip": [
              "127.0.0.1"
            ]
          },
          "netflow": {
            "protocol_identifier": 6,
            "packet_delta_count": 6,
            "vlan_id": 0,
            "flow_start_sys_up_time": 2373708,
            "source_mac_address": "56-E0-32-C1-82-07",
            "octet_delta_count": 312,
            "egress_interface": 4,
            "type": "netflow_flow",
            "destination_ipv4_address": "10.100.11.10",
            "source_ipv4_address": "34.71.180.220",
            "exporter": {
              "uptime_millis": 2449304,
              "address": "127.0.0.1:45715",
              "source_id": 0,
              "version": 9,
              "timestamp": "2024-06-13T23:29:27.000Z"
            },
            "delta_flow_count": 0,
            "tcp_control_bits": 16,
            "ip_class_of_service": 0,
            "ip_version": 4,
            "flow_direction": 0,
            "mpls_label_stack_length": 3,
            "ingress_interface": 3,
            "destination_mac_address": "B4-FB-E4-D0-EA-7B",
            "flow_end_sys_up_time": 2449057,
            "source_transport_port": 443,
            "destination_transport_port": 64047
          },
          "@timestamp": "2024-06-13T23:29:27.000Z",
          "related": {
            "ip": [
              "10.100.11.10",
              "34.71.180.220"
            ]
          },
          "ecs": {
            "version": "8.11.0"
          },
          "data_stream": {
            "namespace": "default",
            "type": "logs",
            "dataset": "netflow.log"
          },
          "host": {
            "hostname": "lima-linux",
            "os": {
              "kernel": "6.8.0-39-generic",
              "codename": "noble",
              "name": "Ubuntu",
              "type": "linux",
              "family": "debian",
              "version": "24.04 LTS (Noble Numbat)",
              "platform": "ubuntu"
            },
            "ip": [
              "192.168.5.15",
              "fe80::5055:55ff:fed3:c3fa",
              "192.168.64.11",
              "fd7a:388c:fc26:8787:5055:55ff:fe57:4f30",
              "fe80::5055:55ff:fe57:4f30",
              "172.17.0.1",
              "172.18.0.1",
              "fe80::42:6eff:fef8:156a",
              "172.19.0.1",
              "fe80::42:23ff:fe61:a7e5",
              "fe80::dc46:b8ff:feeb:7268",
              "fe80::70be:2dff:fef5:9648",
              "fe80::dc52:efff:feac:491c"
            ],
            "containerized": false,
            "name": "lima-linux",
            "id": "2efcdad36f7542db9d7212e71eafeebe",
            "mac": [
              "02-42-23-61-A7-E5",
              "02-42-33-CC-48-A9",
              "02-42-6E-F8-15-6A",
              "52-55-55-57-4F-30",
              "52-55-55-D3-C3-FA",
              "72-BE-2D-F5-96-48",
              "DE-46-B8-EB-72-68",
              "DE-52-EF-AC-49-1C"
            ],
            "architecture": "aarch64"
          },
          "event": {
            "duration": 75349000000,
            "agent_id_status": "auth_metadata_missing",
            "ingested": "2024-07-28T21:31:43Z",
            "created": "2024-07-28T21:31:38.074Z",
            "kind": "event",
            "start": "2024-06-13T23:28:11.404Z",
            "action": "netflow_flow",
            "end": "2024-06-13T23:29:26.753Z",
            "type": [
              "connection"
            ],
            "category": [
              "network"
            ],
            "dataset": "netflow.log"
          },
          "flow": {
            "locality": "external",
            "id": "iZPv2ko_Ghs"
          }
        }
      }
    ]
  }
}
```
