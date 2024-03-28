import metricbeat
import os
import pytest
import sys
import unittest


CONSUL_FIELDS = metricbeat.COMMON_FIELDS + ["consul"]

# raft fields not included here as it's not consistently returned by Consul
AGENT_FIELDS = [
    "runtime.garbage_collector.pause.current.ns",
    "runtime.garbage_collector.pause.total.ns",
    "runtime.garbage_collector.runs",
    "runtime.alloc.bytes",
    "runtime.heap_objects",
    "runtime.malloc_count",
    "runtime.goroutines",
    "runtime.sys.bytes",
]


@metricbeat.parameterized_with_supported_versions
class ConsulAgentTest(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['consul']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_output(self):
        """
        Consul agent module outputs an event.
        """
        self.render_config_template(modules=[{
            "name": "consul",
            "metricsets": ["agent"],
            "hosts": self.get_hosts(),
            "period": "10s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=30)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertCountEqual(self.de_dot(CONSUL_FIELDS), evt.keys())
        consul_agent = evt["consul"]["agent"]

        consul_agent.pop("raft", None)
        consul_agent.pop("autopilot", None)

        print(consul_agent)
        self.assertCountEqual(self.de_dot(AGENT_FIELDS), consul_agent.keys())

        assert(consul_agent["runtime"]["heap_objects"] > 0)

        self.assert_fields_are_documented(evt)
