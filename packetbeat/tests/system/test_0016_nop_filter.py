import unittest
from pbtests.packetbeat import TestCase


class Test(TestCase):
    @unittest.skip(
        "Disabled until filters have been added to processing chain again")
    def test_nop_filter(self):
        """
        Should work fine with the nop filter
        in the configuration file.
        """
        self.render_config_template(
            mysql_ports=[3306],
            filter_plugins=["nop"]
        )

        self.run_packetbeat(pcap="mysql_with_whitespaces.pcap",
                            debug_selectors=["main", "filters"])

        objs = self.read_output()
        assert all([o["type"] == "mysql" for o in objs])
        assert len(objs) == 7

        assert self.log_contains("Filters plugins order: [nop]")

    @unittest.skip(
        "Disabled until filters have been added to processing chain again")
    def test_multiple_nops(self):
        """
        Multiple nops are just as useless as one or none.
        """
        self.render_config_template(
            mysql_ports=[3306],
            filter_plugins=["nop", "nop1", "nop2"],
            filter_config={"nop1": {"type": "nop"}, "nop2": {"type": "nop"}}
        )

        self.run_packetbeat(pcap="mysql_with_whitespaces.pcap",
                            debug_selectors=["main", "filters"])

        objs = self.read_output()
        assert all([o["type"] == "mysql" for o in objs])
        assert len(objs) == 7

        assert self.log_contains("Filters plugins order: [nop nop1 nop2]")
