import metricbeat
import os
import pytest
import redis
import sys
import unittest


REDIS_FIELDS = metricbeat.COMMON_FIELDS + ["redis"]

REDIS_INFO_FIELDS = ["clients", "cluster", "commandstats", "cpu", "memory",
                     "persistence", "replication", "server", "stats", "slowlog"]

REDIS_KEYSPACE_FIELDS = ["keys", "expires", "id", "avg_ttl"]

CPU_FIELDS = ["used.sys", "used.sys_children", "used.user",
              "used.user_children"]

CLIENTS_FIELDS = ["blocked", "connected",
                  "max_input_buffer", "max_output_buffer"]


@metricbeat.parameterized_with_supported_versions
class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['redis']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_info(self):
        """
        Test redis info metricset
        """

        self.render_config_template(modules=[{
            "name": "redis",
            "metricsets": ["info"],
            "hosts": self.get_hosts(),
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        fields = REDIS_FIELDS + ["process", "os"]
        self.assertCountEqual(self.de_dot(fields), evt.keys())
        redis_info = evt["redis"]["info"]
        self.assertCountEqual(self.de_dot(REDIS_INFO_FIELDS), redis_info.keys())
        self.assertCountEqual(self.de_dot(CLIENTS_FIELDS), redis_info["clients"].keys())
        self.assertCountEqual(self.de_dot(CPU_FIELDS), redis_info["cpu"].keys())
        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_keyspace(self):
        """
        Test redis keyspace metricset
        """

        # At least one event must be inserted so db stats exist
        host, port = self.compose_host().split(":")
        r = redis.StrictRedis(
            host=host,
            port=port,
            db=0)
        r.flushall()
        r.set('foo', 'bar')

        self.render_config_template(modules=[{
            "name": "redis",
            "metricsets": ["keyspace"],
            "hosts": self.get_hosts(),
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertCountEqual(self.de_dot(REDIS_FIELDS), evt.keys())
        redis_info = evt["redis"]["keyspace"]
        self.assertCountEqual(self.de_dot(REDIS_KEYSPACE_FIELDS), redis_info.keys())
        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_key(self):
        """
        Test redis key metricset
        """

        # At least one event must be inserted so db stats exist
        host, port = self.compose_host().split(":")
        r = redis.StrictRedis(
            host=host,
            port=port,
            db=0)
        r.flushall()
        r.rpush('list-key', 'one', 'two', 'three')

        self.render_config_template(modules=[{
            "name": "redis",
            "metricsets": ["key"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "additional_content": """
  key.patterns:
    - pattern: list-key
"""
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertCountEqual(self.de_dot(REDIS_FIELDS), evt.keys())
        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_module_processors(self):
        """
        Test local processors for Redis info event.
        """

        fields = ["clients", "cpu"]
        eventFields = ['beat', 'metricset', 'service', 'event']
        eventFields += ['redis.info.' + f for f in fields]
        self.render_config_template(modules=[{
            "name": "redis",
            "metricsets": ["info"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "processors": [{
                "include_fields": eventFields,
            }],
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertCountEqual(self.de_dot(REDIS_FIELDS), evt.keys())
        redis_info = evt["redis"]["info"]
        print(redis_info)
        self.assertCountEqual(fields, redis_info.keys())
        self.assertCountEqual(self.de_dot(CLIENTS_FIELDS), redis_info["clients"].keys())
        self.assertCountEqual(self.de_dot(CPU_FIELDS), redis_info["cpu"].keys())
