import os
import sys
import unittest
import time
from xpack_metricbeat import XPackTest, metricbeat


@unittest.skip("Flaky test: https://github.com/elastic/beats/issues/34993")
class Test(XPackTest):

    COMPOSE_SERVICES = ['oracle']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_query(self):
        """
        sql oracle custom query test
        """
        self.render_config_template(modules=[{
            "name": "sql",
            "metricsets": ["query"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "additional_content": """
  driver: oracle
  sql_query: 'SELECT name, physical_reads, db_block_gets, consistent_gets, 1 - (physical_reads / (db_block_gets + consistent_gets)) FROM V$BUFFER_POOL_STATISTICS'
  sql_response_format: table"""
        }])
        proc = self.start_beat(home=self.beat_path)
        self.wait_until(lambda: self.output_lines() > 0)
        self.wait_until(lambda: self.check_for_events(), max_timeout=300)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        event_valid_counter = False
        for evt in output:
            if evt.get("sql") and evt["sql"].get("query"):
                event_valid_counter = True

    def check_for_events(self):
        output = self.read_output_json()
        for evt in output:
            if evt.get("sql") and evt["sql"].get("query"):
                return True

        return False

    def get_hosts(self):
        return ['user="sys" password="Oradoc_db1" connectString="{}/ORCLPDB1.localdomain" sysdba=true'.format(self.compose_host())]
