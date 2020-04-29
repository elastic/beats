import re
from packetbeat import BaseTest


class Test(BaseTest):
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
        assert o["status"] == "OK"

    def test_mongodb_find_one(self):
        """
        Should correctly pass a simple MongoDB find query.
        The request and response fields should not be in
        by default.
        """
        self.render_config_template(
            mongodb_ports=[27017]
        )
        self.run_packetbeat(pcap="mongo_one_row.pcap",
                            debug_selectors=["mongodb"])

        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "mongodb"
        assert o["method"] == "find"
        assert "request" not in o
        assert "response" not in o

    def test_mongodb_send_response(self):
        """
        Should put the request and the response fields in
        when requested.
        """
        self.render_config_template(
            mongodb_send_request=True,
            mongodb_send_response=True,
            mongodb_ports=[27017]
        )
        self.run_packetbeat(pcap="mongo_one_row.pcap",
                            debug_selectors=["mongodb"])

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]
        assert "request" in o
        assert "response" in o
        assert len(o["response"].splitlines()) == 1
        assert o["source.bytes"] == 50
        assert o["destination.bytes"] == 514

    def test_mongodb_send_response_more_rows(self):
        """
        Should work when the query is returning multiple
        documents.
        """
        self.render_config_template(
            mongodb_send_request=True,
            mongodb_send_response=True,
            mongodb_max_docs=0,
            mongodb_ports=[27017]
        )
        self.run_packetbeat(pcap="mongodb_more_rows.pcap",
                            debug_selectors=["mongodb"])

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]
        assert "request" in o
        assert "response" in o
        assert len(o["response"].splitlines()) == 101

    def test_max_docs_setting(self):
        """
        max_docs setting should be respected.
        """
        self.render_config_template(
            mongodb_send_request=True,
            mongodb_send_response=True,
            mongodb_max_docs=10,
            mongodb_ports=[27017]
        )
        self.run_packetbeat(pcap="mongodb_more_rows.pcap",
                            debug_selectors=["mongodb"])

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]
        assert "request" in o
        assert "response" in o
        assert len(o["response"].splitlines()) == 11

    def test_max_doc_length_setting(self):
        """
        max_doc_length setting should be respected.
        """
        self.render_config_template(
            mongodb_send_request=True,
            mongodb_send_response=True,
            mongodb_max_doc_length=10,
            mongodb_ports=[27017]
        )
        self.run_packetbeat(pcap="mongodb_more_rows.pcap",
                            debug_selectors=["mongodb"])

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]
        assert "request" in o
        assert "response" in o
        # limit lines to 10 chars, but add dots at the end
        assert all([len(l) < 15 for l in o["response"].splitlines()])

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

    def test_session(self):
        """
        Should work for a longer mongodb 3.0 session
        and correctly identify the methods involved.
        """
        self.render_config_template(
            mongodb_ports=[27017]
        )
        self.run_packetbeat(pcap="mongo_3.0_session.pcap",
                            debug_selectors=["mongodb"])

        objs = self.read_output()
        print(len(objs))
        assert len([o for o in objs if o["method"] == "insert"]) == 2
        assert len([o for o in objs if o["method"] == "update"]) == 1
        assert len([o for o in objs if o["method"] == "findandmodify"]) == 1
        assert len([o for o in objs if o["method"] == "listCollections"]) == 5

    def test_write_errors(self):
        """
        Should set status=Error on a bulk write that returns errors.
        """
        self.render_config_template(
            mongodb_ports=[27017]
        )
        self.run_packetbeat(pcap="mongodb_insert_duplicate_key.pcap",
                            debug_selectors=["mongodb"])

        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "mongodb"
        assert o["method"] == "insert"
        assert o["status"] == "Error"
        assert len(o["mongodb.error"]) > 0

    def test_request_after_reply(self):
        """
        Tests that the response time is correctly captured when a single
        reply is seen before the request.
        This is a regression test for bug #216.
        """
        self.render_config_template(
            mongodb_ports=[27017]
        )
        self.run_packetbeat(pcap="mongodb_reply_request_reply.pcap",
                            debug_selectors=["mongodb"])

        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "mongodb"
        assert o["event.duration"] >= 0

    def test_opmsg(self):
        """
        Tests parser works with opcode 2013 (OP_MSG).
        """
        self.render_config_template(
            mongodb_ports=[9991]
        )
        self.run_packetbeat(pcap="mongodb_op_msg_opcode.pcap",
                            debug_selectors=["mongodb"])

        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "mongodb"

        count = self.log_contains_count('Unknown operation code: ')
        assert count == 0

    def test_unknown_opcode_flood(self):
        """
        Tests that any repeated unknown opcodes are reported just once.
        """
        self.render_config_template(
            mongodb_ports=[27017]
        )
        self.run_packetbeat(pcap="mongodb_invalid_opcode_2269.pcap",
                            debug_selectors=["mongodb"])

        unknown_counts = self.log_contains_countmap(
            re.compile(r'Unknown operation code: (\d+)'), 1)

        assert len(unknown_counts) > 0
        for k, v in unknown_counts.items():
            assert v == 1, "Unknown opcode reported more than once: opcode={0}, count={1}".format(
                k, v)
