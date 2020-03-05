import os
import sys
import unittest

from datetime import datetime
from prometheus_pb2 import WriteRequest
import calendar
import requests
import snappy

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))
import metricbeat


PROMETHEUS_FIELDS = metricbeat.COMMON_FIELDS + ["prometheus"]


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['prometheus']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_stats(self):
        """
        prometheus stats test
        """
        self.render_config_template(modules=[{
            "name": "prometheus",
            "metricsets": ["collector"],
            "hosts": self.get_hosts(),
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        evt = output[0]

        self.assertCountEqual(self.de_dot(PROMETHEUS_FIELDS), evt.keys(), evt)

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_remote_write(self):
        """
        prometheus remote_write test
        """
        self.render_config_template(modules=[{
            "name": "prometheus",
            "metricsets": ["remote_write"],
            "period": "5s"
        }])
        proc = self.start_beat()

        self.wait_until(lambda: self.log_contains("Starting HTTP"))

        write_request = WriteRequest()

        series = write_request.timeseries.add()

        # name label always required
        label = series.labels.add()
        label.name = "__name__"
        label.value = "test_metric_name"

        # add labels
        label = series.labels.add()
        label.name = "control_plane_name"
        label.value = "etcd"

        sample = series.samples.add()
        sample.value = 42
        sample.timestamp = self.dt2ts(datetime.utcnow()) * 1000

        uncompressed = write_request.SerializeToString()
        compressed = snappy.compress(uncompressed)

        url = "http://localhost:9201/write"
        headers = {
            "Content-Encoding": "snappy",
            "Content-Type": "application/x-protobuf",
            "X-Prometheus-Remote-Write-Version": "0.1.0",
            "User-Agent": "metrics-worker"
        }
        requests.post(url, headers=headers, data=compressed)

        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        evt = output[0]

        self.assertCountEqual(self.de_dot(PROMETHEUS_FIELDS), evt.keys(), evt)

        self.assert_fields_are_documented(evt)

    def dt2ts(self, dt):
        """Converts a datetime object to UTC timestamp
        naive datetime will be considered UTC.
        """
        return calendar.timegm(dt.utctimetuple())
