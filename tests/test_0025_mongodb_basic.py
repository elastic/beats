from pbtests.packetbeat import TestCase


class Test(TestCase):
    """
    Basic MongoDB tests
    """

    def test_mongodb_use_db(self):
        """
        Should correctly pass a MongoDB database access query
        """
        self.render_config_template(
         mongodb_ports=[27017]
        )
        self.run_packetbeat(pcap="mongodb_use_db.pcap",
                         debug_selectors=["mongodb", "sniffer"])

        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "mongodb"

    def test_mongodb_create_collection(self):
        """
        Should correctly pass a create collection MongoDB database query
        """
        self.render_config_template(
         mongodb_ports=[27017]
        )
        self.run_packetbeat(pcap="mongodb_create_collection.pcap",
                         debug_selectors=["mongodb"])

        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "mongodb"

    def test_mongodb_find(self):
        """
        Should correctly pass a simple MongoDB find query
        """
        self.render_config_template(
            mongodb_ports=[27017]
        )
        self.run_packetbeat(pcap="mongodb_find.pcap",
                            debug_selectors=["mongodb"])

        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "mongodb"
        assert o["method"] == "find"
        assert len(o["mongodb.documents"]) == 1

    def test_mongodb_inserts(self):
        """
        Should correctly pass a MongoDB insert command
        """
        self.render_config_template(
            mongodb_ports=[27017]
        )
        self.run_packetbeat(pcap="mongodb_inserts.pcap",
                            debug_selectors=["mongodb"])

        objs = self.read_output()
        o = objs[1]
        assert o["type"] == "mongodb"
        assert o["method"] == "insert"
