import os
import metricbeat
from nose.plugins.attrib import attr

MYSQL_FIELDS = metricbeat.COMMON_FIELDS + ["mysql"]

MYSQL_STATUS_FIELDS = ["clients", "cluster", "cpu", "keyspace", "memory",
                     "persistence", "replication", "server", "stats"]


class Test(metricbeat.BaseTest):
    @attr('integration')
    def test_status(self):
        """
        MySQL module outputs an event.
        """
        self.render_config_template(modules=[{
            "name": "mysql",
            "metricsets": ["status"],
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

        self.assertItemsEqual(self.de_dot(MYSQL_FIELDS), evt.keys())
        mysql_info = evt["mysql"]["status"]

        self.assert_fields_are_documented(evt)

    def get_hosts(self):
        return [os.getenv('MYSQL_DSN', 'root:test@tcp(localhost:3306)/')]

