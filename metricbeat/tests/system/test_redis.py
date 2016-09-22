import os
import metricbeat
import redis
import unittest
from nose.plugins.attrib import attr

REDIS_FIELDS = metricbeat.COMMON_FIELDS + ["redis"]

REDIS_INFO_FIELDS = ["clients", "cluster", "cpu", "memory",
                     "persistence", "replication", "server", "stats"]

REDIS_KEYSPACE_FIELDS = ["keys", "expires", "id", "avg_ttl"]

CPU_FIELDS = ["used.sys", "used.sys_children", "used.user",
              "used.user_children"]

CLIENTS_FIELDS = ["blocked", "biggest_input_buf",
                  "longest_output_list", "connected"]


class Test(metricbeat.BaseTest):
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

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log, "ERR|WARN")

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertItemsEqual(self.de_dot(REDIS_FIELDS), evt.keys())
        redis_info = evt["redis"]["info"]
        self.assertItemsEqual(self.de_dot(REDIS_INFO_FIELDS), redis_info.keys())
        self.assertItemsEqual(self.de_dot(CLIENTS_FIELDS), redis_info["clients"].keys())
        self.assertItemsEqual(self.de_dot(CPU_FIELDS), redis_info["cpu"].keys())
        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_keysace(self):
        """
        Test redis keyspace metricset
        """

        # At least one event must be inserted so db stats exist
        r = redis.StrictRedis(host=os.getenv('REDIS_HOST', 'localhost'), port=os.getenv('REDIS_PORT', '6379'), db=0)
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

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log, "ERR|WARN")

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertItemsEqual(self.de_dot(REDIS_FIELDS), evt.keys())
        redis_info = evt["redis"]["keyspace"]
        self.assertItemsEqual(self.de_dot(REDIS_KEYSPACE_FIELDS), redis_info.keys())
        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_filters(self):
        """
        Test filters for Redis info event.
        """
        fields = ["clients", "cpu"]
        self.render_config_template(modules=[{
            "name": "redis",
            "metricsets": ["info"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "filters": [{
                "include_fields": fields,
            }],
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log, "ERR|WARN")

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertItemsEqual(self.de_dot(REDIS_FIELDS), evt.keys())
        redis_info = evt["redis"]["info"]
        print redis_info
        self.assertItemsEqual(fields, redis_info.keys())
        self.assertItemsEqual(self.de_dot(CLIENTS_FIELDS), redis_info["clients"].keys())
        self.assertItemsEqual(self.de_dot(CPU_FIELDS), redis_info["cpu"].keys())

    def get_hosts(self):
        return [os.getenv('REDIS_HOST', 'localhost') + ':' +
                os.getenv('REDIS_PORT', '6379')]

