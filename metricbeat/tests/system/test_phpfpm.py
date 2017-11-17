import os
import metricbeat
import unittest

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

        self.assertItemsEqual(self.de_dot(PHPFPM_FIELDS), evt.keys(), evt)

        self.assert_fields_are_documented(evt)

    def get_hosts(self):
        return [os.getenv('PHPFPM_HOST', 'localhost') + ':' +
                os.getenv('PHPFPM_PORT', '81')]
