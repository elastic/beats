from packetbeat import BaseTest

"""
Tests for SMTP
"""


class Test(BaseTest):

    def test_smtp_basic_data(self):
        """
        Should parse basic SMTP session
        """
        self.render_config_template(
            smtp_ports=['25'],
            smtp_send_request=False,
            smtp_send_response=False,
            smtp_send_data_headers=True,
            smtp_send_data_body=True,
        )

        self.run_packetbeat(pcap="smtp_basic_data.pcap",
                            debug_selectors=["smtp"])
        objs = self.read_output()

        assert len(objs) == 4
        assert all([o["type"] == "smtp" for o in objs])
        assert all([o["status"] == "OK" for o in objs])

        assert objs[0]["smtp.type"] == "PROMPT"
        assert objs[0]["smtp.response.code"] == 220
        assert objs[1]["smtp.type"] == "COMMAND"
        assert objs[1]["smtp.request.command"] == "EHLO"
        assert objs[1]["smtp.response.code"] == 250
        assert objs[1]["smtp.response.phrases"][4] == "PRDR"
        assert objs[2]["smtp.type"] == "MAIL"
        assert "smtp.session_id" in objs[2]
        assert objs[2]["smtp.envelope_sender"] == "bar@example.org"
        assert objs[2]["smtp.envelope_recipients"][0] == "foo@example.org"
        assert objs[2]["smtp.headers"]["Subject"] == "Test"
        assert objs[2]["smtp.body"] == "Testing\r\n"
        assert objs[3]["smtp.request.command"] == "QUIT"
        assert objs[3]["smtp.response.code"] == 221
        assert objs[3]["smtp.response.phrases"][0] == "localhost closing connection"

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
        # Filter out ICMP messages
        objs = [o for o in objs if o["type"] != "icmp"]

        assert len(objs) == 7
        assert all([o["type"] == "smtp" for o in objs])
        assert all([o["status"] == "OK" for o in objs])
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
        assert "smtp.headers" in objs[2]
        d = objs[2]["smtp.headers"]
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

        assert len(objs) == 3
        assert all([o["type"] == "smtp" for o in objs])
        assert all([o["status"] == "OK" for o in objs])

        # "MAIL" is the last transaction (before packet loss)
        assert objs[-1]["smtp.type"] == "MAIL"
        assert objs[-1]["notes"][0] == "Packet loss while capturing the transaction"

    def test_smtp_tcp_gap_in_response(self):
        """
        Should recover if there's enough data to resync after TCP gap
        in response
        """
        self.render_config_template(
            smtp_ports=['25'],
            smtp_send_request=False,
            smtp_send_response=False,
            smtp_send_data_headers=True,
            smtp_send_data_body=True,
        )

        self.run_packetbeat(pcap="smtp_tcp_gap_in_response.pcap",
                            debug_selectors=["smtp"])
        objs = self.read_output()

        assert len(objs) == 5
        assert all([o["type"] == "smtp" for o in objs])
        assert all([o["status"] == "OK" for o in objs])
        assert objs[2]["smtp.type"] == "MAIL"
        assert "smtp.envelope_sender" in objs[2]
        assert "smtp.headers" not in objs[2]
        assert "smtp.body" not in objs[2]
        assert objs[2]["notes"][0] == "Packet loss while capturing the transaction"
        assert objs[3]["smtp.type"] == "MAIL"
        assert "smtp.envelope_sender" not in objs[3]
        assert "smtp.headers" in objs[3]
        assert "smtp.body" in objs[3]
        # "QUIT" is the last request (after recovery from packet loss)
        assert objs[-1]["smtp.request.command"] == "QUIT"
        assert objs[-1]["smtp.response.code"] == 221

    def test_smtp_tcp_gap_no_220_prompt(self):
        """
        Should skip request the beginning of a session
        """
        self.render_config_template(
            smtp_ports=['25'],
            smtp_send_request=True,
            smtp_send_response=True,
            smtp_send_data_headers=True,
            smtp_send_data_body=True,
        )

        self.run_packetbeat(pcap="smtp_tcp_gap_no_220_prompt.pcap",
                            debug_selectors=["smtp"])
        objs = self.read_output()

        assert len(objs) == 2
        assert all([o["type"] == "smtp" for o in objs])
        assert all([o["status"] == "OK" for o in objs])

        assert objs[0]["smtp.type"] == "MAIL"
        assert "smtp.envelope_sender" in objs[0]
        assert "smtp.envelope_recipients" in objs[0]
        assert "smtp.headers" in objs[0]
        assert "smtp.body" in objs[0]
        assert objs[-1]["smtp.request.command"] == "QUIT"
        assert objs[-1]["smtp.response.code"] == 221

    def test_smtp_incomplete_prompt_response(self):
        """
        Should skip mangled (incomplete) 220 prompt response
        """
        self.render_config_template(
            smtp_ports=['25'],
            smtp_send_request=True,
            smtp_send_response=True,
            smtp_send_data_headers=True,
            smtp_send_data_body=True,
        )

        self.run_packetbeat(pcap="smtp_incomplete_prompt_response.pcap",
                            debug_selectors=["smtp"])
        objs = self.read_output()

        assert len(objs) == 3
        assert all([o["type"] == "smtp" for o in objs])
        assert all([o["status"] == "OK" for o in objs])

        # "EHLO" command is the first request
        assert objs[0]["smtp.request.command"] == "EHLO"
        # Multiline 250 is the first response
        assert objs[0]["smtp.response.code"] == 250
        assert objs[0]["smtp.response.phrases"][4] == "PRDR"
        # "QUIT" command is the last request
        assert objs[-1]["smtp.request.command"] == "QUIT"
        # 221 code is the last response
        assert objs[-1]["smtp.response.code"] == 221

    def test_smtp_multiple_message(self):
        """
        Should parse SMTP sessions with multiple messages
        """
        self.render_config_template(
            smtp_ports=['25'],
            smtp_send_request=True,
            smtp_send_response=True,
            smtp_send_data_headers=True,
            smtp_send_data_body=True,
        )

        self.run_packetbeat(pcap="smtp_multiple_messages.pcap",
                            debug_selectors=["smtp"])
        objs = self.read_output()

        assert len(objs) == 5
        assert all([o["type"] == "smtp" for o in objs])
        assert all([o["status"] == "OK" for o in objs])

        assert objs[0]["smtp.response.code"] == 220
        assert objs[1]["smtp.request.command"] == "EHLO"
        assert objs[1]["smtp.response.phrases"][4] == "PRDR"
        assert objs[2]["smtp.type"] == "MAIL"
        assert "smtp.headers" in objs[2]
        assert "smtp.body" in objs[2]
        assert "smtp.envelope_sender" in objs[2]
        assert "smtp.envelope_recipients" in objs[2]
        assert objs[3]["smtp.type"] == "MAIL"
        assert "smtp.headers" in objs[3]
        assert "smtp.body" in objs[3]
        assert "smtp.envelope_sender" in objs[3]
        assert "smtp.envelope_recipients" in objs[3]
        assert objs[4]["smtp.request.command"] == "QUIT"
        assert objs[4]["smtp.response.code"] == 221

    def test_smtp_multiple_late_start(self):
        """
        Should recover from late start parsing multiple messages
        """
        self.render_config_template(
            smtp_ports=['25'],
            smtp_send_request=False,
            smtp_send_response=False,
            smtp_send_data_headers=True,
            smtp_send_data_body=True,
        )

        self.run_packetbeat(pcap="smtp_multiple_late_start.pcap",
                            debug_selectors=["smtp"])
        objs = self.read_output()

        assert len(objs) == 3
        assert all([o["type"] == "smtp" for o in objs])
        assert all([o["status"] == "OK" for o in objs])

        assert objs[0]["smtp.type"] == "MAIL"
        assert "smtp.headers" in objs[0]
        assert "smtp.body" in objs[0]
        assert "smtp.envelope_sender" not in objs[0]
        assert "smtp.envelope_recipients" not in objs[0]
        assert objs[1]["smtp.type"] == "MAIL"
        assert "smtp.headers" in objs[1]
        assert "smtp.body" in objs[1]
        assert "smtp.envelope_sender" in objs[1]
        assert "smtp.envelope_recipients" in objs[1]
        assert objs[2]["smtp.request.command"] == "QUIT"
        assert objs[2]["smtp.response.code"] == 221

    def test_smtp_multiple_late_data_transmission(self):
        """
        Should recover from late start with multiple messages in the
        middle of data transmission
        """
        self.render_config_template(
            smtp_ports=['25'],
            smtp_send_request=True,
            smtp_send_response=True,
            smtp_send_data_headers=True,
            smtp_send_data_body=True,
        )

        self.run_packetbeat(pcap="smtp_multiple_late_start.pcap",
                            debug_selectors=["smtp"])
        objs = self.read_output()

        assert len(objs) == 3
        assert all([o["type"] == "smtp" for o in objs])
        assert all([o["status"] == "OK" for o in objs])

        assert objs[0]["smtp.type"] == "MAIL"
        assert "smtp.headers" in objs[0]
        assert "smtp.body" in objs[0]
        assert "smtp.envelope_sender" not in objs[0]
        assert "smtp.envelope_recipients" not in objs[0]
        assert objs[1]["smtp.type"] == "MAIL"
        assert "smtp.headers" in objs[1]
        assert "smtp.body" in objs[1]
        assert "smtp.envelope_sender" in objs[1]
        assert "smtp.envelope_recipients" in objs[1]
        assert objs[2]["smtp.request.command"] == "QUIT"
        assert objs[2]["smtp.response.code"] == 221

    def test_smtp_multiple_late_smtp_data(self):
        """
        Should not get confused by embedded SMTP transcript in the
        data of a late start session
        """
        self.render_config_template(
            smtp_ports=['25'],
            smtp_send_request=False,
            smtp_send_response=False,
            smtp_send_data_headers=True,
            smtp_send_data_body=True,
        )

        self.run_packetbeat(pcap="smtp_multiple_late_embedded_smtp.pcap",
                            debug_selectors=["smtp"])
        objs = self.read_output()

        assert len(objs) == 3
        assert all([o["type"] == "smtp" for o in objs])
        assert all([o["status"] == "OK" for o in objs])

        assert objs[0]["smtp.type"] == "MAIL"
        assert "smtp.headers" not in objs[0]
        assert "smtp.body" not in objs[0]
        assert "smtp.envelope_sender" not in objs[0]
        assert "smtp.envelope_recipients" not in objs[0]
        assert "notes" in objs[0]
        assert objs[1]["smtp.type"] == "MAIL"
        assert "smtp.headers" in objs[1]
        assert "smtp.body" in objs[1]
        assert "smtp.envelope_sender" in objs[1]
        assert "smtp.envelope_recipients" in objs[1]
        assert objs[2]["smtp.request.command"] == "QUIT"
        assert objs[2]["smtp.response.code"] == 221

    def test_smtp_split_requests(self):
        """
        Should not get confused by multiple segment requests
        """
        self.render_config_template(
            smtp_ports=['25'],
            smtp_send_request=False,
            smtp_send_response=False,
            smtp_send_data_headers=True,
            smtp_send_data_body=True,
        )

        self.run_packetbeat(pcap="smtp_tcp_split_during_sync.pcap",
                            debug_selectors=["smtp"])
        objs = self.read_output()

        assert len(objs) == 4
        assert all([o["type"] == "smtp" for o in objs])
        assert all([o["status"] == "OK" for o in objs])

        assert objs[0]["smtp.type"] == "PROMPT"
        assert objs[1]["smtp.type"] == "COMMAND"
        assert objs[2]["smtp.type"] == "MAIL"
        assert objs[3]["smtp.type"] == "COMMAND"
