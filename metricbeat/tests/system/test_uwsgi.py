import os
import logging
import metricbeat
import unittest
from nose.plugins.attrib import attr

logger = logging.getLogger(__name__)


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['uwsgi_tcp', "uwsgi_http"]

    def common_checks(self, output):
        # Ensure no errors or warnings exist in the log.
        self.assert_no_logged_warnings()

        cores = []
        total = None
        workers = []

        for evt in output:
            top_level_fields = metricbeat.COMMON_FIELDS + ["uwsgi"]
            self.assertItemsEqual(self.de_dot(top_level_fields), evt.keys())

            self.assert_fields_are_documented(evt)

            if "total" in evt["uwsgi"]["status"]:
                total = evt["uwsgi"]["status"]["total"]

            if "core" in evt["uwsgi"]["status"]:
                cores.append(evt["uwsgi"]["status"]["core"])

            if "worker" in evt["uwsgi"]["status"]:
                workers.append(evt["uwsgi"]["status"]["worker"])

        requests = 0
        for core in cores:
            requests += core["requests"]["total"]

        assert requests == total["requests"]
        assert requests > 0

        assert len(workers) > 0
        assert len(cores) > 0

        assert "accepting" in workers[0]
        assert "worker_pid" in cores[0]
        assert "requests" in cores[0]
        assert "static" in cores[0]["requests"]

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_status_tcp(self):
        """
        uWSGI module outputs an event.
        """
        hosts = [os.getenv("UWSGI_STAT_TCP_SERVER")]
        self.render_config_template(modules=[{
            "name": "uwsgi",
            "metricsets": ["status"],
            "hosts": hosts,
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        output = self.read_output_json()
        self.common_checks(output)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_status_http(self):
        """
        uWSGI module outputs an event.
        """
        hosts = [os.getenv("UWSGI_STAT_HTTP_SERVER")]
        self.render_config_template(modules=[{
            "name": "uwsgi",
            "metricsets": ["status"],
            "hosts": hosts,
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        output = self.read_output_json()
        self.common_checks(output)
