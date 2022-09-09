import os
import sys
import unittest
from xpack_metricbeat import XPackTest, metricbeat


class Test(XPackTest):

    COMPOSE_SERVICES = ['mysql']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_query(self):
        """
        sql custom query test
        """
        self.render_config_template(modules=[{
            "name": "sql",
            "metricsets": ["query"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "additional_content": """
  driver: mysql
  sql_response_format: variables
  sql_query: 'select table_schema, table_name, engine, table_rows from information_schema.tables where table_rows > 0'"""
        }])
        proc = self.start_beat(home=self.beat_path)
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            self.assertIn("sql", evt.keys(), evt)
            self.assertIn("query", evt["sql"].keys(), evt)

    def get_hosts(self):
        return ['root:test@tcp({})/'.format(self.compose_host())]
