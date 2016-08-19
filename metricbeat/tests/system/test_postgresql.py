import os
import metricbeat
from nose.plugins.attrib import attr


class Test(metricbeat.BaseTest):

    def common_checks(self, output):
        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log, "ERR|WARN")

        for evt in output:
            top_level_fields = metricbeat.COMMON_FIELDS + ["postgresql"]
            self.assertItemsEqual(self.de_dot(top_level_fields), evt.keys())

            self.assert_fields_are_documented(evt)

    def get_hosts(self):
        return [os.getenv("POSTGRESQL_DSN")]

    @attr('integration')
    def test_activity(self):
        """
        PostgreSQL module outputs an event.
        """
        self.render_config_template(modules=[{
            "name": "postgresql",
            "metricsets": ["activity"],
            "hosts": self.get_hosts(),
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        output = self.read_output_json()
        self.common_checks(output)

        for evt in output:
            assert "name" in evt["postgresql"]["activity"]["database"]
            assert "oid" in evt["postgresql"]["activity"]["database"]
            assert "state" in evt["postgresql"]["activity"]

    @attr('integration')
    def test_database(self):
        """
        PostgreSQL module outputs an event.
        """
        self.render_config_template(modules=[{
            "name": "postgresql",
            "metricsets": ["database"],
            "hosts": self.get_hosts(),
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        output = self.read_output_json()
        self.common_checks(output)

        for evt in output:
            assert "name" in evt["postgresql"]["database"]
            assert "oid" in evt["postgresql"]["database"]
            assert "blocks" in evt["postgresql"]["database"]
            assert "rows" in evt["postgresql"]["database"]
            assert "conflicts" in evt["postgresql"]["database"]
            assert "deadlocks" in evt["postgresql"]["database"]
