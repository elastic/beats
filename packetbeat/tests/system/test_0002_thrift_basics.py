from packetbeat import BaseTest


class Test(BaseTest):

    def tutorial_asserts(self, objs):
        assert len(objs) == 17
        assert all([o["type"] == "thrift" for o in objs])

        # bytes_in present for all. bytes_out present for
        # all except the zip async calls.
        assert all([o["bytes_in"] > 0 for o in objs])
        assert all([o["bytes_out"] > 0 for o in objs[0:14]])
        assert all([o["bytes_out"] > 0 for o in objs[16:]])
        assert objs[14]["bytes_out"] == 0
        assert objs[15]["bytes_out"] == 0

        assert objs[0]["method"] == "ping"
        assert objs[0]["thrift.params"] == "()"
        assert objs[0]["thrift.return_value"] == ""

        assert objs[1]["method"] == "add"
        assert objs[1]["thrift.params"] == "(1: 1, 2: 1)"
        assert objs[1]["thrift.return_value"] == "2"

        assert objs[2]["method"] == "add16"
        assert objs[2]["query"] == "add16(1: 1, 2: 1)"
        assert objs[2]["thrift.params"] == "(1: 1, 2: 1)"
        assert objs[2]["thrift.return_value"] == "2"

        assert objs[3]["method"] == "add64"
        assert objs[3]["query"] == "add64(1: 1, 2: 1)"
        assert objs[3]["thrift.params"] == "(1: 1, 2: 1)"
        assert objs[3]["thrift.return_value"] == "2"

        assert objs[4]["method"] == "add_doubles"
        assert objs[4]["thrift.params"] == "(1: 1.2, 2: 1.3)"
        assert objs[4]["thrift.return_value"] == "2.5"

        assert objs[5]["method"] == "echo_bool"
        assert objs[5]["thrift.params"] == "(1: true)"
        assert objs[5]["thrift.return_value"] == "true"

        assert objs[6]["method"] == "echo_string"
        assert objs[6]["thrift.params"] == "(1: \"hello\")"
        assert objs[6]["thrift.return_value"] == "\"hello\""

    def test_thrift_tutorial_socket(self):
        self.render_config_template(
            thrift_ports=[9090],
            thrift_send_request=True,
            thrift_send_response=True,
        )
        self.run_packetbeat(pcap="thrift_tutorial.pcap",
                            debug_selectors=["thrift"])

        objs = self.read_output()

        self.tutorial_asserts(objs)

        assert all([len(o["request"]) > 0 for o in objs])
        assert objs[0]["request"] == "ping()"
        assert objs[11]["response"] == "Exceptions: (1: (1: 4, 2: " + \
            "\"Cannot divide by 0\"))"
        assert all([o["port"] == 9090 for o in objs])

    def test_send_options_default(self):
        """
            Request and response should be off by default.
        """
        self.render_config_template(
            thrift_ports=[9090],
        )
        self.run_packetbeat(pcap="thrift_tutorial.pcap",
                            debug_selectors=["thrift"])

        objs = self.read_output()

        self.tutorial_asserts(objs)

        assert all(["request" not in o for o in objs])
        assert all(["response" not in o for o in objs])

    def test_thrift_tutorial_framed(self):
        self.render_config_template(
            thrift_ports=[9090],
            thrift_transport_type="framed"
        )
        self.run_packetbeat(pcap="thrift_tutorial_framed_transport.pcap",
                            debug_selectors=["thrift"])

        objs = self.read_output()

        self.tutorial_asserts(objs)

    def test_thrift_tutorial_with_idl(self):
        self.render_config_template(
            thrift_ports=[9090],
            thrift_idl_files=["tutorial.thrift", "shared.thrift"]
        )
        self.copy_files(["tutorial.thrift", "shared.thrift"])
        self.run_packetbeat(pcap="thrift_tutorial.pcap",
                            debug_selectors=["thrift"])

        objs = self.read_output()
        assert len(objs) == 17
        assert all([o["type"] == "thrift" for o in objs])
        assert all([o["thrift.service"] == "Calculator" or
                    o["thrift.service"] == "SharedService" for o in objs])

        assert objs[0]["method"] == "ping"
        assert objs[0]["thrift.params"] == "()"
        assert objs[0]["thrift.return_value"] == ""

        assert objs[1]["method"] == "add"
        assert objs[1]["thrift.params"] == "(num1: 1, num2: 1)"
        assert objs[1]["thrift.return_value"] == "2"

        assert objs[2]["method"] == "add16"
        assert objs[2]["thrift.params"] == "(num1: 1, num2: 1)"
        assert objs[2]["thrift.return_value"] == "2"

        assert objs[3]["method"] == "add64"
        assert objs[3]["thrift.params"] == "(num1: 1, num2: 1)"
        assert objs[3]["thrift.return_value"] == "2"

        assert objs[4]["method"] == "add_doubles"
        assert objs[4]["thrift.params"] == \
            "(num1: 1.2, num2: 1.3)"
        assert objs[4]["thrift.return_value"] == "2.5"

        assert objs[5]["method"] == "echo_bool"
        assert objs[5]["thrift.params"] == "(b: true)"
        assert objs[5]["thrift.return_value"] == "true"

        assert objs[6]["method"] == "echo_string"
        assert objs[6]["thrift.params"] == "(s: \"hello\")"
        assert objs[6]["thrift.return_value"] == "\"hello\""

    def test_thrift_integration(self):
        """
        Test based on the integration test suite of the Thrift
        project. Pcap generated by running:

        py/TestServer.py --proto=binary --port=9090 \
            --genpydir=gen-py TSimpleServer
        py/TestClient.py --proto=binary --port=9090 \
            --host=localhost --genpydir=gen-py
        """

        self.render_config_template(
            thrift_ports=[9090],
            thrift_idl_files=["ThriftTest.thrift"]
        )

        self.copy_files(["ThriftTest.thrift"])
        self.run_packetbeat(pcap="thrift_integration.pcap",
                            debug_selectors=["thrift"])

        objs = self.read_output()
        assert len(objs) == 26
        assert all([o["type"] == "thrift" for o in objs])
        assert all([o["thrift.service"] == "ThriftTest" for o in objs])

        # check a few things

        assert objs[0]["method"] == "testByte"
        assert objs[0]["thrift.params"] == "(thing: 63)"
        assert objs[0]["thrift.return_value"] == "63"

        assert objs[5]["method"] == "testEnum"
        assert objs[5]["thrift.params"] == "(thing: 5)"
        assert objs[5]["thrift.return_value"] == "5"

        assert objs[17]["method"] == "testOneway"
        assert objs[17]["thrift.params"] == "(secondsToSleep: 1)"
        assert "thrift.return_value" not in objs[17]
        assert objs[17]["bytes_out"] == 0

        assert objs[21]["method"] == "testString"
        assert objs[21]["thrift.params"] == "(thing: \"" + \
            ("Python" * 20) + "\")"
        assert objs[21]["thrift.return_value"] == '"' + \
            ("Python" * 20) + '"'

    def test_thrift_send_request_response(self):
        # send_request=true send_response=false
        self.render_config_template(
            thrift_ports=[9090],
            thrift_idl_files=["ThriftTest.thrift"],
            thrift_send_request=True,
            thrift_send_response=False,
        )
        self.copy_files(["ThriftTest.thrift"])
        self.run_packetbeat(pcap="thrift_integration.pcap",
                            debug_selectors=["thrift"])

        objs = self.read_output()

        assert all([len(o["request"]) > 0 for o in objs])
        assert all(["response" not in o for o in objs])

        # send_request=false send_response=false
        self.render_config_template(
            thrift_ports=[9090],
            thrift_idl_files=["ThriftTest.thrift"],
            thrift_no_send_request=True,
            thrift_no_send_response=True,
        )
        self.copy_files(["ThriftTest.thrift"])
        self.run_packetbeat(pcap="thrift_integration.pcap",
                            debug_selectors=["thrift"])

        objs = self.read_output()

        assert all(["request" not in o for o in objs])
        assert all(["response" not in o for o in objs])

    def test_thrift_binary(self):
        self.render_config_template(
            thrift_ports=[9090],
            thrift_transport_type="framed",
            thrift_idl_files=["tutorial.thrift", "shared.thrift"]
        )
        self.copy_files(["tutorial.thrift", "shared.thrift"])
        self.run_packetbeat(pcap="thrift_echo_binary.pcap",
                            debug_selectors=["thrift"])

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["method"] == "echo_binary"
        assert o["thrift.return_value"] == "ab0c1d281a000000"
