import os
import sys
import unittest
import time
from xpack_metricbeat import XPackTest, metricbeat


class Test(XPackTest):
    COMPOSE_SERVICES = ['mssql']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_query_without_fetch_from_all_databases(self):
        """
        SQL MSSQL custom query with fetch_from_all_databases=false

        1 document will be received from the default selected database: 'master' in MSSQL.
        """
        self.run_query_test(fetch_from_all_databases=False, expected_output_count=1)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_query_with_fetch_from_all_databases(self):
        """
        SQL MSSQL custom query with fetch_from_all_databases=true

        4 documents will be received, each corresponding to one of the four default databases in MSSQL.
        """
        self.run_query_test(fetch_from_all_databases=True, expected_output_count=4)

    def run_query_test(self, fetch_from_all_databases: bool, expected_output_count: int) -> None:
        self.render_config_template(modules=[{
            "name": "sql",
            "metricsets": ["query"],
            "hosts": ['sqlserver://{}:{}@{}'.format(self.get_username(), self.get_password(), self.compose_host())],
            "period": "5s",
            "additional_content": f"""
  driver: mssql
  fetch_from_all_databases: {str(fetch_from_all_databases).lower()}
  sql_query: SELECT DB_NAME() AS 'database_name';
  sql_response_format: table"""
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_count() >= expected_output_count, max_timeout=60)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()

        try:
            self.assertEqual(len(output), expected_output_count,
                            f"Expected {expected_output_count} documents, got {len(output)}")

            expected_databases = ["master", "model", "msdb", "tempdb"]
            found_databases = set()

            for evt in output:
                self.assert_fields_are_documented(evt)
                self.assertIn("sql", evt, "Event is missing 'sql' key")
                self.assertIn("query", evt["sql"], "Event is missing 'query' key in 'sql' object")
                self.assertIn("database_name", evt["sql"]["query"], "Event is missing 'database_name' in query results")

                db_name = evt["sql"]["query"]["database_name"]
                self.assertIn(db_name, expected_databases, f"Unexpected database name: {db_name}")
                found_databases.add(db_name)

            if fetch_from_all_databases:
                self.assertEqual(found_databases, set(expected_databases),
                                f"Not all expected databases were found. Missing: {set(expected_databases) - found_databases}")
            else:
                self.assertEqual(len(found_databases), 1, "Expected only one database when fetch_from_all_databases is False")
                self.assertIn(list(found_databases)[0], expected_databases, "The single database should be one of the expected databases")

        except AssertionError as e:
            print(f"Test failed. Output received: {output}")
            raise e


    def get_username(self):
        return os.getenv('MSSQL_USERNAME', 'SA')

    def get_password(self):
        return os.getenv('MSSQL_PASSWORD', '1234_asdf')
