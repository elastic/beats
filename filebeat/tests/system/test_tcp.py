from filebeat import BaseTest
import socket


class Test(BaseTest):
    """
    Test filebeat TCP input
    """

    def test_tcp_with_newline_delimiter(self):
        """
        Test TCP input with a new line delimiter
        """
        self.send_events_with_delimiter("\n")

    def test_tcp_with_custom_char_delimiter(self):
        """
        Test TCP input with a custom single char delimiter
        """
        self.send_events_with_delimiter(";")

    def test_tcp_with_custom_word_delimiter(self):
        """
        Test TCP input with a custom single char delimiter
        """
        self.send_events_with_delimiter("<END>")

    def test_tcp_with_rfc6587_non_transparent(self):
        """
        Test TCP input with rfc6587 non_transparent framing
        """
        self.send_events_with_rfc6587_framing("non-transparent")

    def test_tcp_with_rfc6587_octet(self):
        """
        Test TCP input with rfc6587 octet counting framing
        """
        self.send_events_with_rfc6587_framing("octet")

    def send_events_with_delimiter(self, delimiter):
        host = "127.0.0.1"
        port = 8080
        input_raw = """
- type: tcp
  host: "{}:{}"
  enabled: true
"""

        # Use default of \n and stripping \r
        if delimiter != "":
            input_raw += "\n  line_delimiter: {}".format(delimiter)

        input_raw = input_raw.format(host, port)

        self.render_config_template(
            input_raw=input_raw,
            inputs=False,
        )

        filebeat = self.start_beat()

        self.wait_until(lambda: self.log_contains("Start accepting connections"))

        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)  # TCP
        sock.connect((host, port))

        for n in range(0, 2):
            sock.send(bytes("Hello World: " + str(n) + delimiter, "utf-8"))

        self.wait_until(lambda: self.output_count(lambda x: x >= 2))

        filebeat.check_kill_and_wait()

        output = self.read_output()

        assert len(output) == 2
        assert output[0]["input.type"] == "tcp"

        sock.close()

    def send_events_with_rfc6587_framing(self, framing):
        host = "127.0.0.1"
        port = 8080
        delimiter = "\n"
        input_raw = """
- type: tcp
  host: "{}:{}"
  enabled: true
  framing: rfc6587
"""

        input_raw += "\n  line_delimiter: {}".format(delimiter)

        input_raw = input_raw.format(host, port)

        self.render_config_template(
            input_raw=input_raw,
            inputs=False,
        )

        filebeat = self.start_beat()

        self.wait_until(lambda: self.log_contains("Start accepting connections"))

        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)  # TCP
        sock.connect((host, port))

        for n in range(0, 2):
            if framing == "non-transparent":
                sock.send(bytes("Hello World: " + str(n) + "\n", "utf-8"))
            if framing == "octet":
                sock.send(bytes("14 Hello World: " + str(n), "utf-8"))

        self.wait_until(lambda: self.output_count(lambda x: x >= 2))

        filebeat.check_kill_and_wait()

        output = self.read_output()

        assert len(output) == 2
        assert output[0]["input.type"] == "tcp"

        sock.close()
