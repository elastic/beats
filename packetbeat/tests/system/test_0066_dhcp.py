from packetbeat import BaseTest

"""
Tests for the DHCPv4 protocol.
"""


class Test(BaseTest):

    def test_dhcp(self):
        self.render_config_template()
        self.run_packetbeat(pcap="dhcp.pcap")

        objs = self.read_output(types=['dhcpv4'])
        assert len(objs) == 4

        assert objs[0]["client_ip"] == "0.0.0.0"
        assert objs[0]["client_port"] == 68
        assert objs[0]["dhcpv4.client_mac"] == "00:0b:82:01:fc:42"
        assert objs[0]["dhcpv4.flags"] == "unicast"
        assert objs[0]["dhcpv4.hardware_type"] == "Ethernet"
        assert objs[0]["dhcpv4.hops"] == 0
        assert objs[0]["dhcpv4.op_code"] == "bootrequest"
        assert objs[0]["dhcpv4.option.message_type"] == "discover"
        assert objs[0]["dhcpv4.option.parameter_request_list"] == [
            "Subnet Mask",
            "Router",
            "Domain Name Server",
            "NTP Servers"
        ]
        assert objs[0]["dhcpv4.option.requested_ip_address"] == "0.0.0.0"
        assert objs[0]["dhcpv4.seconds"] == 0
        assert objs[0]["dhcpv4.transaction_id"] == "0x00003d1d"
        assert objs[0]["ip"] == "255.255.255.255"
        assert objs[0]["port"] == 67
        assert objs[0]["status"] == "OK"
        assert objs[0]["transport"] == "udp"
        assert objs[0]["type"] == "dhcpv4"

        assert objs[1]["client_ip"] == "192.168.0.10"
        assert objs[1]["client_port"] == 68
        assert objs[1]["dhcpv4.assigned_ip"] == "192.168.0.10"
        assert objs[1]["dhcpv4.client_mac"] == "00:0b:82:01:fc:42"
        assert objs[1]["dhcpv4.flags"] == "unicast"
        assert objs[1]["dhcpv4.hardware_type"] == "Ethernet"
        assert objs[1]["dhcpv4.hops"] == 0
        assert objs[1]["dhcpv4.op_code"] == "bootreply"
        assert objs[1]["dhcpv4.option.ip_address_lease_time_sec"] == 3600
        assert objs[1]["dhcpv4.option.message_type"] == "offer"
        assert objs[1]["dhcpv4.option.rebinding_time_sec"] == 3150
        assert objs[1]["dhcpv4.option.renewal_time_sec"] == 1800
        assert objs[1]["dhcpv4.option.server_identifier"] == "192.168.0.1"
        assert objs[1]["dhcpv4.option.subnet_mask"] == "255.255.255.0"
        assert objs[1]["dhcpv4.seconds"] == 0
        assert objs[1]["dhcpv4.transaction_id"] == "0x00003d1d"
        assert objs[1]["ip"] == "192.168.0.1"
        assert objs[1]["port"] == 67
        assert objs[1]["status"] == "OK"
        assert objs[1]["transport"] == "udp"
        assert objs[1]["type"] == "dhcpv4"

        assert objs[2]["client_ip"] == "0.0.0.0"
        assert objs[2]["client_port"] == 68
        assert objs[2]["dhcpv4.client_mac"] == "00:0b:82:01:fc:42"
        assert objs[2]["dhcpv4.flags"] == "unicast"
        assert objs[2]["dhcpv4.hardware_type"] == "Ethernet"
        assert objs[2]["dhcpv4.hops"] == 0
        assert objs[2]["dhcpv4.op_code"] == "bootrequest"
        assert objs[2]["dhcpv4.option.message_type"] == "request"
        assert objs[2]["dhcpv4.option.parameter_request_list"] == [
            "Subnet Mask",
            "Router",
            "Domain Name Server",
            "NTP Servers"
        ]
        assert objs[2]["dhcpv4.option.requested_ip_address"] == "192.168.0.10"
        assert objs[2]["dhcpv4.option.server_identifier"] == "192.168.0.1"
        assert objs[2]["dhcpv4.seconds"] == 0
        assert objs[2]["dhcpv4.transaction_id"] == "0x00003d1e"
        assert objs[2]["ip"] == "255.255.255.255"
        assert objs[2]["port"] == 67
        assert objs[2]["status"] == "OK"
        assert objs[2]["transport"] == "udp"
        assert objs[2]["type"] == "dhcpv4"

        assert objs[3]["client_ip"] == "192.168.0.10"
        assert objs[3]["client_port"] == 68
        assert objs[3]["dhcpv4.assigned_ip"] == "192.168.0.10"
        assert objs[3]["dhcpv4.client_mac"] == "00:0b:82:01:fc:42"
        assert objs[3]["dhcpv4.flags"] == "unicast"
        assert objs[3]["dhcpv4.hardware_type"] == "Ethernet"
        assert objs[3]["dhcpv4.hops"] == 0
        assert objs[3]["dhcpv4.op_code"] == "bootreply"
        assert objs[3]["dhcpv4.option.ip_address_lease_time_sec"] == 3600
        assert objs[3]["dhcpv4.option.message_type"] == "ack"
        assert objs[3]["dhcpv4.option.rebinding_time_sec"] == 3150
        assert objs[3]["dhcpv4.option.renewal_time_sec"] == 1800
        assert objs[3]["dhcpv4.option.server_identifier"] == "192.168.0.1"
        assert objs[3]["dhcpv4.option.subnet_mask"] == "255.255.255.0"
        assert objs[3]["dhcpv4.seconds"] == 0
        assert objs[3]["dhcpv4.transaction_id"] == "0x00003d1e"
        assert objs[3]["ip"] == "192.168.0.1"
        assert objs[3]["port"] == 67
        assert objs[3]["status"] == "OK"
        assert objs[3]["transport"] == "udp"
        assert objs[3]["type"] == "dhcpv4"
