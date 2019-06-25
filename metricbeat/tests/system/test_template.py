import os
import metricbeat
import json
from nose.plugins.skip import SkipTest


class Test(metricbeat.BaseTest):

    def test_export_template(self):
        """
        Test export template works and contains all fields
        """

        if os.name == "nt":
            raise SkipTest

        self.render_config_template("metricbeat",
                                    os.path.join(self.working_dir,
                                                 "metricbeat.yml"),
                                    )

        # Remove fields.yml to make sure template is built from internal binary data
        os.remove(os.path.join(self.working_dir, "fields.yml"))

        exit_code = self.run_beat(
            logging_args=[],
            extra_args=["export", "template"],
            config="metricbeat.yml",
            output="template.json"
        )
        assert exit_code == 0

        template_path = os.path.join(self.working_dir, "template.json")
        template_content = ""

        # Read in all json lines and discard the coverage info
        with open(template_path) as f:
            for line in f:
                template_content += line
                if line.startswith("}"):
                    break

        t = json.loads(template_content)
        properties = t["mappings"]["properties"]

        # Check libbeat fields
        assert properties["@timestamp"] == {"type": "date"}
        assert properties["host"]["properties"]["name"] == {"type": "keyword", "ignore_above": 1024}

        # Check metricbeat generic field
        assert properties["service"]["properties"]["address"] == {"type": "keyword", "ignore_above": 1024}

        # Check module specific field
        assert properties["system"]["properties"]["cpu"]["properties"]["cores"] == {"type": "long"}
