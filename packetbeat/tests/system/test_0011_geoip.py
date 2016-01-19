from packetbeat import BaseTest
import os

"""
Tests for reading the geoip files.
"""


class Test(BaseTest):

    def test_geoip_config_disabled(self):
        self.render_config_template(
            http_ports=[8002],
            http_real_ip_header="X-Forward-For",
            http_send_all_headers=True,
            geoip_paths=[]
        )
        self.run_packetbeat(pcap="http_realip.pcap", debug_selectors=["http"])

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["real_ip"] == "89.247.39.104"
        assert "client_location" not in o

    def test_geoip_config_from_file(self):
        self.render_config_template(
            http_ports=[8002],
            http_real_ip_header="X-Forward-For",
            http_send_all_headers=True,
            geoip_paths=["geoip_city.dat"]
        )
        # geoip_onrange.dat is generated from geoip_onerange.csv
        # by using https://github.com/mteodoro/mmutils
        self.copy_files(["geoip_city.dat"])
        self.run_packetbeat(pcap="http_realip.pcap", debug_selectors=["http"])

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["real_ip"] == "89.247.39.104"
        assert o["client_location"] == "52.528503, 13.410904"

    def test_geoip_symlink(self):
        """
        Should be able to follow symlinks to GeoIP libs.
        """
        self.render_config_template(
            http_ports=[8002],
            http_real_ip_header="X-Forward-For",
            http_send_all_headers=True,
            geoip_paths=["geoip.dat"]
        )
        self.copy_files(["geoip_city.dat"])
        os.symlink("geoip_city.dat",
                   os.path.join(self.working_dir, "geoip.dat"))

        self.run_packetbeat(pcap="http_realip.pcap", debug_selectors=["http"])

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["real_ip"] == "89.247.39.104"
        assert o["client_location"] == "52.528503, 13.410904"
