from packetbeat import BaseTest

"""
Tests for SMTP
"""


class Test(BaseTest):

    def test_smtp_basic_test(self):
        """
        Should parse basic SMTP session
        """
        self.render_config_template(
            smtp_ports=['25'],
            smtp_send_request=False,
            smtp_send_response=False,
            smtp_send_data_headers=False,
            smtp_send_data_body=False,
        )

        self.run_packetbeat(pcap="smtp_basic_data.pcap",
                            debug_selectors=["smtp"])
        objs = self.read_output()

        assert len(objs) == 7
        assert all([o["type"] == "smtp" for o in objs])
        assert all([o["status"] == "OK" for o in objs])
        assert all(["smtp.response.code" in o for o in objs])

        assert objs[0]["smtp.response.code"] == 220
        assert objs[1]["smtp.request.command"] == "EHLO"
        assert objs[1]["smtp.response.code"] == 250
        assert objs[1]["smtp.response.phrases"][4] == "PRDR"
        assert objs[2]["smtp.request.command"] == "MAIL"
        assert objs[2]["smtp.response.code"] == 250
        assert objs[2]["smtp.response.phrases"][0] == "OK"
        assert objs[3]["smtp.request.command"] == "RCPT"
        assert objs[3]["smtp.response.code"] == 250
        assert objs[3]["smtp.response.phrases"][0] == "Accepted"
        assert objs[4]["smtp.request.command"] == "DATA"
        assert objs[4]["smtp.response.code"] == 354
        assert objs[5]["smtp.request.command"] == "EOD"
        assert objs[5]["smtp.response.code"] == 250
        assert objs[6]["smtp.request.command"] == "QUIT"
        assert objs[6]["smtp.response.code"] == 221

    def test_smtp_auth_login(self):
        """
        Should fall back to handling AUTH credentials as request commands/args
        """
        self.render_config_template(
            # Disable DNS (the pcap contains a DNS query)
            dns_ports=[],
            smtp_ports=['25'],
            smtp_send_request=False,
            smtp_send_response=False,
            smtp_send_data_headers=False,
            smtp_send_data_body=True,
        )

        self.run_packetbeat(pcap="smtp_auth_login.pcap",
                            debug_selectors=["smtp"])
        objs = self.read_output()
        assert len(objs) == 14
        # Filter out ICMP messages
        objs = objs[:8] + objs[12:]

        assert all([o["type"] == "smtp" for o in objs])
        assert all([o["status"] == "OK" for o in objs])
        assert all(["smtp.response.code" in o for o in objs])
        assert all([o["smtp.response.code"] < 400 for o in objs])
        assert objs[3]["smtp.request.command"].startswith("Z3V")
        assert objs[3]["smtp.response.code"] == 334
        assert objs[3]["smtp.response.phrases"][0] == "UGFzc3dvcmQ6"

    def test_smtp_data_attachment(self):
        """
        Should parse RFC 5322 data payloads
        """
        self.render_config_template(
            smtp_ports=['25'],
            smtp_send_request=True,
            smtp_send_response=False,
            smtp_send_data_headers=True,
            smtp_send_data_body=True,
        )

        self.run_packetbeat(pcap="smtp_attachment.pcap",
                            debug_selectors=["smtp"])
        objs = self.read_output()

        assert all([o["type"] == "smtp" for o in objs])
        assert all([o["status"] == "OK" for o in objs])
        assert all(["smtp.response.code" in o for o in objs])
        assert "smtp.request.headers" in objs[5]
        d = objs[5]["smtp.request.headers"]
        assert "Content-Type" in d
        assert d["Content-Type"].startswith("multipart/mixed")

    def test_smtp_tcp_gap_in_request(self):
        """
        Should generate no new transactions if there's not enough data to
        resync after TCP gap in request
        """
        self.render_config_template(
            smtp_ports=['25'],
            smtp_send_request=True,
            smtp_send_response=True,
            smtp_send_data_headers=False,
            smtp_send_data_body=True,
        )

        self.run_packetbeat(pcap="smtp_tcp_gap_in_data_request.pcap",
                            debug_selectors=["smtp"])
        objs = self.read_output()

        assert len(objs) == 5
        assert all([o["type"] == "smtp" for o in objs])
        assert all([o["status"] == "OK" for o in objs])
        assert all(["smtp.response.code" in o for o in objs])

        # "DATA" is the last request (before packet loss)
        assert objs[-1]["smtp.request.command"] == "DATA"
        # "Enter message" is the last response (before packet loss)
        assert objs[-1]["smtp.response.code"] == 354

    def test_smtp_tcp_gap_in_response(self):
        """
        Should recover if there's enough data to resync after TCP gap
        in response
        """
        self.render_config_template(
            smtp_ports=['25'],
            smtp_send_request=False,
            smtp_send_response=False,
            smtp_send_data_headers=False,
            smtp_send_data_body=False,
        )

        self.run_packetbeat(pcap="smtp_tcp_gap_in_response.pcap",
                            debug_selectors=["smtp"])
        objs = self.read_output()

        assert len(objs) == 5
        assert all([o["type"] == "smtp" for o in objs])
        assert all([o["status"] == "OK" for o in objs])
        assert all(["smtp.response.code" in o for o in objs])

        # "QUIT" is the last request command (after recovery from packet loss)
        assert objs[-1]["smtp.request.command"] == "QUIT"
        # 221 is the last response code (after recovery from packet loss)
        assert objs[-1]["smtp.response.code"] == 221

    def test_smtp_tcp_gap_no_220_prompt(self):
        """
        Should handle lack of a 220 prompt in the beginning of a session
        """
        self.render_config_template(
            smtp_ports=['25'],
            smtp_send_request=False,
            smtp_send_response=False,
            smtp_send_data_headers=False,
            smtp_send_data_body=False,
        )

        self.run_packetbeat(pcap="smtp_tcp_gap_no_220_prompt.pcap",
                            debug_selectors=["smtp"])
        objs = self.read_output()

        assert len(objs) == 5
        assert all([o["type"] == "smtp" for o in objs])
        assert all([o["status"] == "OK" for o in objs])
        assert all(["smtp.response.code" in o for o in objs])

        # "MAIL" command is the first request
        assert objs[0]["smtp.request.command"] == "MAIL"
        # 250 code is the first response
        assert objs[0]["smtp.response.code"] == 250
        assert objs[0]["smtp.response.phrases"][0] == "OK"
        # "QUIT" command is the last request
        assert objs[-1]["smtp.request.command"] == "QUIT"
        # 221 code is the last response
        assert objs[-1]["smtp.response.code"] == 221

    def test_smtp_incomplete_prompt_response(self):
        """
        Should skip mangled (incomplete) 220 prompt response
        """
        self.render_config_template(
            smtp_ports=['25'],
            smtp_send_request=False,
            smtp_send_response=False,
            smtp_send_data_headers=False,
            smtp_send_data_body=False,
        )

        self.run_packetbeat(pcap="smtp_incomplete_prompt_response.pcap",
                            debug_selectors=["smtp"])
        objs = self.read_output()

        assert len(objs) == 6
        assert all([o["type"] == "smtp" for o in objs])
        assert all([o["status"] == "OK" for o in objs])
        assert all(["smtp.response.code" in o for o in objs])

        # "EHLO" command is the first request
        assert objs[0]["smtp.request.command"] == "EHLO"
        # Multiline 250 is the first response
        assert objs[0]["smtp.response.code"] == 250
        assert objs[0]["smtp.response.phrases"][4] == "PRDR"
        # "QUIT" command is the last request
        assert objs[-1]["smtp.request.command"] == "QUIT"
        # 221 code is the last response
        assert objs[-1]["smtp.response.code"] == 221
