import os
import sys
import unittest

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))
import metricbeat


PHPFPM_FIELDS = metricbeat.COMMON_FIELDS + ["php_fpm"]


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['phpfpm']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_info(self):
        """
        php_fpm pool metricset test
        """
        self.render_config_template(modules=[{
            "name": "php_fpm",
            "metricsets": ["pool"],
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

        self.assertCountEqual(self.de_dot(PHPFPM_FIELDS), evt.keys(), evt)

        self.assert_fields_are_documented(evt)
