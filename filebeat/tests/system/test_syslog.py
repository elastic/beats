from filebeat import BaseTest
import socket


class Test(BaseTest):
    """
    Test filebeat with the syslog input
    """

    def test_syslog_with_tcp(self):
        """
        Test syslog input with events from TCP.
        """
        host = "127.0.0.1"
        port = 8080
        input_raw = """
- type: syslog
  protocol:
    tcp:
        host: "{}:{}"
"""

        input_raw = input_raw.format(host, port)
        self.render_config_template(
            input_raw=input_raw,
            inputs=False,
        )

        filebeat = self.start_beat()

        self.wait_until(lambda: self.log_contains("Started listening for TCP connection"))

        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)  # TCP
        sock.connect((host, port))

        for n in range(0, 2):
            m = "<13>Oct 11 22:14:15 wopr.mymachine.co postfix/smtpd[2000]:" \
                " 'su root' failed for lonvick on /dev/pts/8 {}\n"
            m = m.format(n)
            sock.send(m)

        self.wait_until(lambda: self.output_count(lambda x: x >= 2))

        filebeat.check_kill_and_wait()

        output = self.read_output()

        assert len(output) == 2
        self.assert_syslog(output[0])
        sock.close()

    def test_syslog_with_udp(self):
        """
        Test syslog input with events from TCP.
        """
        host = "127.0.0.1"
        port = 8080
        input_raw = """
- type: syslog
  protocol:
    udp:
        host: "{}:{}"
"""

        input_raw = input_raw.format(host, port)
        self.render_config_template(
            input_raw=input_raw,
            inputs=False,
        )

        filebeat = self.start_beat()

        self.wait_until(lambda: self.log_contains("Started listening for UDP connection"))

        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)  # TCP

        for n in range(0, 2):
            m = "<13>Oct 11 22:14:15 wopr.mymachine.co postfix/smtpd[2000]:" \
                " 'su root' failed for lonvick on /dev/pts/8 {}\n"
            m = m.format(n)
            sock.sendto(m, (host, port))

        self.wait_until(lambda: self.output_count(lambda x: x >= 2))

        filebeat.check_kill_and_wait()

        output = self.read_output()

        assert len(output) == 2
        self.assert_syslog(output[0])

    def assert_syslog(self, syslog):
        assert syslog["event.severity"] == 5
        assert syslog["hostname"] == "wopr.mymachine.co"
        assert syslog["input.type"] == "syslog"
        assert syslog["message"] == "'su root' failed for lonvick on /dev/pts/8 0"
        assert syslog["process.pid"] == 2000
        assert syslog["process.program"] == "postfix/smtpd"
        assert syslog["syslog.facility"] == 1
        assert syslog["syslog.priority"] == 13
        assert syslog["syslog.severity_label"] == "Notice"
        assert syslog["syslog.facility_label"] == "user-level"
        assert len(syslog["log.source.address"]) > 0
