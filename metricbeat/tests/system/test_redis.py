import os
import metricbeat
import redis
import unittest
from nose.plugins.attrib import attr

REDIS_FIELDS = metricbeat.COMMON_FIELDS + ["redis"]

REDIS_INFO_FIELDS = ["clients", "cluster", "cpu", "memory",
                     "persistence", "replication", "server", "stats", "slowlog"]

REDIS_KEYSPACE_FIELDS = ["keys", "expires", "id", "avg_ttl"]

CPU_FIELDS = ["used.sys", "used.sys_children", "used.user",
              "used.user_children"]

CLIENTS_FIELDS = ["blocked", "biggest_input_buf",
                  "longest_output_list", "connected",
                  "max_input_buffer", "max_output_buffer"]


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['redis']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @attr('integration')
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
        self.assertItemsEqual(self.de_dot(fields), evt.keys())
        redis_info = evt["redis"]["info"]
        self.assertItemsEqual(self.de_dot(REDIS_INFO_FIELDS), redis_info.keys())
        self.assertItemsEqual(self.de_dot(CLIENTS_FIELDS), redis_info["clients"].keys())
        self.assertItemsEqual(self.de_dot(CPU_FIELDS), redis_info["cpu"].keys())
        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @attr('integration')
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

        self.assertItemsEqual(self.de_dot(REDIS_FIELDS), evt.keys())
        redis_info = evt["redis"]["keyspace"]
        self.assertItemsEqual(self.de_dot(REDIS_KEYSPACE_FIELDS), redis_info.keys())
        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @attr('integration')
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

        self.assertItemsEqual(self.de_dot(REDIS_FIELDS), evt.keys())
        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @attr('integration')
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

        self.assertItemsEqual(self.de_dot(REDIS_FIELDS), evt.keys())
        redis_info = evt["redis"]["info"]
        print(redis_info)
        self.assertItemsEqual(fields, redis_info.keys())
        self.assertItemsEqual(self.de_dot(CLIENTS_FIELDS), redis_info["clients"].keys())
        self.assertItemsEqual(self.de_dot(CPU_FIELDS), redis_info["cpu"].keys())


class TestRedis4(Test):
    COMPOSE_SERVICES = ['redis_4']


class TestRedis5(Test):
    COMPOSE_SERVICES = ['redis_5']
