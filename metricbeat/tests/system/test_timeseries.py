import os
import shutil
import metricbeat
import multiprocessing


class TestTimeseries(metricbeat.BaseTest):
    """
    Test metricbeat timeseries.instance generation
    """

    def test_enable_timeseries(self):
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["cpu"],
            "period": "5s"
        }])

        proc = self.start_beat(extra_args=["-E", "timeseries.enabled=true"])
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]
        self.assert_fields_are_documented(evt)

        assert 'instance' in evt['timeseries']
