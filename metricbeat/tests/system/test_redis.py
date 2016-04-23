import os
import metricbeat
from nose.plugins.attrib import attr

REDIS_FIELDS = metricbeat.COMMON_FIELDS + ["redis-info"]

REDIS_INFO_FIELDS = ["clients", "cluster", "cpu", "keyspace", "memory",
                     "persistence", "replication", "server", "stats"]

CPU_FIELDS = ["used_cpu_sys", "used_cpu_sys_children", "used_cpu_user",
              "used_cpu_user_children"]

CLIENTS_FIELDS = ["blocked_clients", "client_biggest_input_buf",
                  "client_longest_output_list", "connected_clients"]


class RedisInfoTest(metricbeat.BaseTest):
    @attr('integration')
    def test_output(self):
        """
        Redis module outputs an event.
        """
        self.render_config_template(modules=[{
            "name": "redis",
            "metricsets": ["info"],
            "hosts": [os.getenv('REDIS_HOST') + ":6379"],
        }])
        proc = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1)
        )
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log, "ERR|WARN")

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertItemsEqual(REDIS_FIELDS, evt.keys())
        redis_info = evt["redis-info"]
        self.assertItemsEqual(REDIS_INFO_FIELDS, redis_info.keys())
        self.assertItemsEqual(CLIENTS_FIELDS, redis_info["clients"].keys())
        self.assertItemsEqual(CPU_FIELDS, redis_info["cpu"].keys())

        # TODO: After fields.yml is updated this can be uncommented.
        #self.assert_fields_are_documented(evt)

    @attr('integration')
    def test_filters(self):
        """
        Test filters for Redis info event.
        """
        fields = ["clients", "cpu"]
        self.render_config_template(modules=[{
            "name": "redis",
            "metricsets": ["info"],
            "hosts": [os.getenv('REDIS_HOST') + ":6379"],
            "filters": [{
                "include_fields": fields,
            }],
        }])
        proc = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1)
        )
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log, "ERR|WARN")

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertItemsEqual(REDIS_FIELDS, evt.keys())
        redis_info = evt["redis-info"]
        self.assertItemsEqual(fields, redis_info.keys())
        self.assertItemsEqual(CLIENTS_FIELDS, redis_info["clients"].keys())
        self.assertItemsEqual(CPU_FIELDS, redis_info["cpu"].keys())
