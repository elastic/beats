from packetbeat import BaseTest

import six


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
        assert all([o["event.dataset"] == "redis" for o in objs])

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
        assert all([isinstance(o["method"], six.string_types) for o in objs[3:]])
        assert all([isinstance(o["resource"], six.string_types) for o in objs[3:]])
        assert all([isinstance(o["query"], six.string_types) for o in objs[3:]])

        assert all(["source.bytes" in o for o in objs])
        assert all(["destination.bytes" in o for o in objs])

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

        assert all([isinstance(o["source.bytes"], int) for o in objs])
        assert all([isinstance(o["destination.bytes"], int) for o in objs])
        assert all([o["source.bytes"] > 0 for o in objs])
        assert all([o["destination.bytes"] > 0 for o in objs])
