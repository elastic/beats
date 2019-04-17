import os
import metricbeat
import unittest
from nose.plugins.attrib import attr

MYSQL_FIELDS = metricbeat.COMMON_FIELDS + ["mysql"]

MYSQL_STATUS_FIELDS = ["clients", "cluster", "cpu", "keyspace", "memory",
                       "persistence", "replication", "server", "stats"]


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['mysql']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
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
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertItemsEqual(self.de_dot(MYSQL_FIELDS), evt.keys(), evt)

        status = evt["mysql"]["status"]
        assert status["connections"] > 0
        assert status["opened_tables"] > 0

        self.assert_fields_are_documented(evt)

    def get_hosts(self):
        return ['root:test@tcp({}:3306)/'.format(self.compose_hosts()[0])]


class TestMysql80(Test):
    COMPOSE_SERVICES = ['mysql_8_0']


class TestPercona57(Test):
    COMPOSE_SERVICES = ['percona_5_7']


class TestPercona80(Test):
    COMPOSE_SERVICES = ['percona_8_0']


class TestMariadb102(Test):
    COMPOSE_SERVICES = ['mariadb_10_2']


class TestMariadb103(Test):
    COMPOSE_SERVICES = ['mariadb_10_3']
