from base import BaseTest
import os


class Test(BaseTest):

    def test_export_index_pattern(self):
        """
        Test export index pattern
        """
        self.render_config_template("mockbeat",
                                    os.path.join(self.working_dir,
                                                 "mockbeat.yml"),
                                    fields=os.path.join(self.working_dir, "fields.yml"))
        exit_code = self.run_beat(
            logging_args=[],
            extra_args=["export", "index-pattern"],
            config="mockbeat.yml")

        assert exit_code == 0
        assert self.log_contains('"type": "index-pattern"')
