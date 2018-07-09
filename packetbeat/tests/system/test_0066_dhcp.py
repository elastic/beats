from packetbeat import BaseTest

"""
Tests for the DHCP protocol.
"""


class Test(BaseTest):

    def test_dhcp(self):
        self.render_config_template(
            dhcp_ports=[67],
        )
        self.run_packetbeat(pcap="dhcp.pcap")

        objs = self.read_output(types=['dhcp'])
        assert len(objs) == 4
        assert objs[0]['dhcp.client_ip'] == ''
        assert objs[0]['dhcp.server_ip'] == ''
        assert objs[0]['dhcp.op_code'] == 1
        assert objs[0]['dhcp.hops'] == 0
        assert objs[0]['dhcp.gateway_ip'] == ''
        assert objs[0]['dhcp.client_hwaddr'] == '00:0b:82:01:fc:42'
        assert objs[0]['dhcp.message_type'] == 'DHCPDISCOVER'
        assert objs[0]['dhcp.transaction_id'] == '00003d1d'
        assert objs[0]['dhcp.server_name'] == ''
        assert objs[0]['dhcp.hardware_type'] == 'ethernet'
        assert objs[0]['dhcp.assigned_ip'] == ''
        assert objs[1]['dhcp.server_ip'] == '192.168.0.1'
        assert objs[1]['dhcp.assigned_ip'] == '192.168.0.10'
        assert objs[1]['dhcp.server_identifier'] == '192.168.0.1'
        assert objs[1]['dhcp.message_type'] == 'DHCPOFFER'
