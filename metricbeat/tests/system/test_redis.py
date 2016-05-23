import os
import metricbeat
from nose.plugins.attrib import attr

REDIS_FIELDS = metricbeat.COMMON_FIELDS + ["redis"]

REDIS_INFO_FIELDS = ["clients", "cluster", "cpu", "keyspace", "memory",
                     "persistence", "replication", "server", "stats"]

CPU_FIELDS = ["used.sys", "used.sys_children", "used.user",
              "used.user_children"]

CLIENTS_FIELDS = ["blocked", "biggest_input_buf",
                  "longest_output_list", "connected"]


class Test(metricbeat.BaseTest):
    @attr('integration')
    def test_output(self):
        """
        Redis module outputs an event.
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

        # Delete keyspace entry as this one is dynamic
        del evt["redis"]["info"]["keyspace"]

        self.assert_fields_are_documented(evt)

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

