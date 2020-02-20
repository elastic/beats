import os
import socket
import sys

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))
from xpack_metricbeat import XPackTest, metricbeat

STATSD_HOST = '127.0.0.1'
STATSD_PORT = 8125

METRIC_MESSAGE = bytes('metric1:777.0|g|#k1:v1,k2:v2', 'utf-8')


class Test(XPackTest):

    def test_server(self):
        """
        statsd server metricset test
        """

        # Start the application
        self.render_config_template(modules=[{
            "name": "statsd",
            "metricsets": ["server"],
            "period": "5s",
            "host": STATSD_HOST,
            "port": STATSD_PORT,
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("Started listening for UDP"))

        # Send UDP packet with metric
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.sendto(METRIC_MESSAGE, (STATSD_HOST, STATSD_PORT))
        sock.close()

        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings(replace='use of closed network connection')

        # Verify output
        output = self.read_output_json()
        self.assertGreater(len(output), 0)
        evt = output[0]
        assert evt["statsd"]["metric1"]["value"] == 777
        self.assert_fields_are_documented(evt)
