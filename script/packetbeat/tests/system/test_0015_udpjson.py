from packetbeat import BaseTest
from nose.tools import nottest
import socket


class Test(BaseTest):

    @nottest
    def test_udpjson_config(self):
        """
        Should start with sniffer and udpjson inputs configured.
        """
        self.render_config_template(
            mysql_ports=[3306],
            input_plugins=["sniffer", "udpjson"]
        )

        self.run_packetbeat(pcap="mysql_with_whitespaces.pcap")

        objs = self.read_output()
        assert all([o["type"] == "mysql" for o in objs])
        assert len(objs) == 7

    @nottest
    def test_only_udpjson_config(self):
        """
        It should be possible to start without the sniffer configured.
        """
        self.render_config_template(
            input_plugins=["udpjson"]
        )

        packetbeat = self.start_packetbeat(debug_selectors=["udpjson"])

        self.wait_until(
            lambda: self.log_contains(
                msg="UDPJson plugin listening on 127.0.0.1:9712"),
            max_timeout=2)

        packetbeat.kill_and_wait()

    @nottest
    def test_send_udpjson_msg(self):
        """
        It should be possible to send a UDP message and read it from
        the output.
        """
        self.render_config_template(
            input_plugins=["udpjson"]
        )

        packetbeat = self.start_packetbeat(debug_selectors=["udpjson"])
        self.wait_until(
            lambda: self.log_contains(
                msg="UDPJson plugin listening on 127.0.0.1:9712"),
            max_timeout=2,
            name="Log contains listening")

        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.sendto("""{"type": "http", "status": "OK"}""",
                    ("127.0.0.1", 9712))
        sock.sendto("""{"type": "mysql", "status": "Error"}""",
                    ("127.0.0.1", 9712))

        self.wait_until(
            lambda: self.output_has(lines=2),
            max_timeout=2,
            name="Output has 2 lines")

        objs = self.read_output()

        expected_fields = ["shipper", "status", "type", "@timestamp", "count"]
        self.all_have_only_fields(objs, expected_fields)

        assert objs[0]["type"] == "http"
        assert objs[0]["status"] == "OK"

        assert objs[1]["type"] == "mysql"
        assert objs[1]["status"] == "Error"

        packetbeat.kill_and_wait()
