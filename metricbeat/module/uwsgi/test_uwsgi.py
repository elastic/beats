import logging
import os
import sys
import unittest
from nose.plugins.attrib import attr
from parameterized import parameterized

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))
import metricbeat


logger = logging.getLogger(__name__)


@metricbeat.parameterized_with_supported_versions
class Test(metricbeat.BaseTest):
    COMPOSE_SERVICES = ['uwsgi_http', 'uwsgi_tcp']

    def common_checks(self, output):
        # Ensure no errors or warnings exist in the log.
        self.assert_no_logged_warnings()

        cores = []
        total = None
        workers = []

        for evt in output:
            top_level_fields = metricbeat.COMMON_FIELDS + ["uwsgi"]
            self.assertCountEqual(self.de_dot(top_level_fields), evt.keys())

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

    @parameterized.expand(["http", "tcp"])
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_status(self, proto):
        """
        uWSGI module outputs an event.
        """
        self.render_config_template(modules=[{
            "name": "uwsgi",
            "metricsets": ["status"],
            "hosts": [self.get_host(proto)],
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        output = self.read_output_json()
        self.common_checks(output)

    def get_host(self, proto):
        return proto + "://" + self.compose_host(service="uwsgi_"+proto)
