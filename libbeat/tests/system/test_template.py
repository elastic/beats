from base import BaseTest

import os
import json


class Test(BaseTest):

    def test_generate_templates(self):
        """
        Generates templates from other Beats.
        """
        self.render_config_template()

        output_json = os.path.join(self.working_dir, "template.json")
        fields_yml = "../../../../fields.yml"

        exit_code = self.run_beat(extra_args=[
            "-E", "setup.template.output_to_file.path={}".format(output_json),
            "-E", "setup.template.fields={}".format(fields_yml)])
        assert exit_code == 1

        # check json file
        with open(output_json) as f:
            tmpl = json.load(f)
        assert "mappings" in tmpl

    def test_generate_templates_v5(self):
        """
        Generates templates from other Beats.
        """
        self.render_config_template()

        output_json = os.path.join(self.working_dir, "template-5x.json")
        fields_yml = "../../../../fields.yml"

        exit_code = self.run_beat(extra_args=[
            "-E", "setup.template.output_to_file.path={}".format(output_json),
            "-E", "setup.template.output_to_file.version=5.0.0".format(output_json),
            "-E", "setup.template.fields={}".format(fields_yml)])
        assert exit_code == 1

        # check json file
        with open(output_json) as f:
            tmpl = json.load(f)
        assert "mappings" in tmpl
        assert tmpl["mappings"]["_default_"]["_all"]["norms"]["enabled"] is False
