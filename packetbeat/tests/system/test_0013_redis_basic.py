from packetbeat import BaseTest


class Test(BaseTest):
    """
    Basic REDIS tests
    """

    def test_redis_session(self):
        """
        Should correctly pass a simple Redis Session containing
        also an error.
        """
        self.render_config_template(
            redis_ports=[6380]
        )
        self.run_packetbeat(pcap="redis_session.pcap", debug_selectors=["*"])

        objs = self.read_output()
        assert all([o["type"] == "redis" for o in objs])

        assert objs[0]["method"] == "SET"
        assert objs[0]["resource"] == "key3"
        assert objs[0]["query"] == "set key3 me"
        assert objs[0]["status"] == "OK"
        assert objs[0]["redis.return_value"] == "OK"

        assert objs[1]["status"] == "OK"
        assert objs[1]["method"] == "GET"
        assert objs[1]["redis.return_value"] == "me"
        assert objs[1]["query"] == "get key3"
        assert objs[1]["redis.return_value"] == "me"

        assert objs[2]["status"] == "Error"
        assert objs[2]["method"] == "LLEN"
        assert objs[2]["redis.error"] == "ERR Operation against a key " + \
            "holding the wrong kind of value"

        # the rest should be successful
        assert all([o["status"] == "OK" for o in objs[3:]])
        assert all(["redis.return_value" in o for o in objs[3:]])
        assert all([isinstance(o["method"], basestring) for o in objs[3:]])
        assert all([isinstance(o["resource"], basestring) for o in objs[3:]])
        assert all([isinstance(o["query"], basestring) for o in objs[3:]])

        assert all(["bytes_in" in o for o in objs])
        assert all(["bytes_out" in o for o in objs])

    def test_byteout_bytein(self):
        """
        Should have non-zero byte_in and byte_out values.
        """
        self.render_config_template(
            redis_ports=[6380]
        )
        self.run_packetbeat(pcap="redis_session.pcap")

        objs = self.read_output()
        assert all([o["type"] == "redis" for o in objs])

        assert all([isinstance(o["bytes_out"], int) for o in objs])
        assert all([isinstance(o["bytes_in"], int) for o in objs])
        assert all([o["bytes_out"] > 0 for o in objs])
        assert all([o["bytes_in"] > 0 for o in objs])
