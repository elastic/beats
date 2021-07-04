import os
import socket
import sys
from xpack_metricbeat import XPackTest, metricbeat

STATSD_HOST = '127.0.0.1'
STATSD_PORT = 8126

METRIC_MESSAGE = bytes('dagrun.duration.failed.a_dagid:200|ms|#k1:v1,k2:v2', 'utf-8')


class Test(XPackTest):

    def test_server(self):
        """
        airflow statsd metricset test
        """

        # Start the application
        self.render_config_template(modules=[{
            "name": "airflow",
            "metricsets": ["statsd"],
            "period": "5s",
            "host": STATSD_HOST,
            "port": STATSD_PORT,
        }])
        proc = self.start_beat(home=self.beat_path)
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

        del evt["airflow"]["dag_duration"]["mean_rate"]  # floating
        del evt["airflow"]["dag_duration"]["1m_rate"]  # floating
        del evt["airflow"]["dag_duration"]["5m_rate"]  # floating
        del evt["airflow"]["dag_duration"]["15m_rate"]  # floating

        assert evt["airflow"]["dag_id"] == "a_dagid"
        assert evt["airflow"]["status"] == "failure"
        assert evt["airflow"]["dag_duration"] == {
            "p99_9": 200,
            "count": 1,
            "median": 200,
            "p99": 200,
            "p95": 200,
            "min": 200,
            "stddev": 0,
            "p75": 200,
            "max": 200,
            "mean": 200,
        }
        self.assert_fields_are_documented(evt)
