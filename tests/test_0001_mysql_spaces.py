from pbtests.packetbeat import TestCase


class Test(TestCase):
    def test_mysql_with_spaces(self):
        self.render_config_template(
            mysql_ports=[3306]
        )
        self.run_packetbeat(pcap="mysql_with_whitespaces.pcap")
