import json
import unittest
import yaml

from beat.beat import INTEGRATION_TESTS


class TestExportsMixin:

    def run_export_cmd(self, cmd, extra=[]):
        """
        Runs the given export command and returns the output as a string.
        Raises an exception if the command fails.
        :param cmd: the export command
        :param extra: Extra arguments (optional)
        :return: The output as a string.
        """
        self.render_config_template()

        args = ["export", cmd]
        if len(extra) != 0:
            args += extra
        exit_code = self.run_beat(extra_args=args, logging_args=[])
        output = self.get_log()
        if exit_code != 0:
            raise Exception("export command returned with an error: {}".format(output))
        trailer = "\nPASS\n"
        pos = output.rfind(trailer)
        if pos == -1:
            raise Exception("didn't return expected trailer:{} got:{}".format(
                trailer.__repr__(),
                output[-100:].__repr__()))
        return output[:pos]

    def test_export_ilm_policy(self):
        """
        Test that the ilm-policy can be exported with `export ilm-policy`
        """
        output = self.run_export_cmd("ilm-policy")
        js = json.loads(output)
        assert "policy" in js

    def test_export_template(self):
        """
        Test that the template can be exported with `export template`
        """
        output = self.run_export_cmd("template")
        js = json.loads(output)
        assert "index_patterns" in js and "mappings" in js

    def test_export_index_pattern(self):
        """
        Test that the index-pattern can be exported with `export index-pattern`
        """
        output = self.run_export_cmd("index-pattern")
        js = json.loads(output)
        assert "objects" in js
        size = len(output.encode('utf-8'))
        assert size < 1024 * 1024, "Kibana index pattern must be less than 1MiB " \
            "to keep the Beat setup request size below " \
            "Kibana's server.maxPayloadBytes."

    def test_export_index_pattern_migration(self):
        """
        Test that the index-pattern can be exported with `export index-pattern` (migration enabled)
        """
        output = self.run_export_cmd("index-pattern", extra=['-E', 'migration.6_to_7.enabled=true'])
        js = json.loads(output)
        assert "objects" in js
        size = len(output.encode('utf-8'))
        assert size < 1024 * 1024, "Kibana index pattern must be less than 1MiB " \
            "to keep the Beat setup request size below " \
            "Kibana's server.maxPayloadBytes."

    def test_export_config(self):
        """
        Test that the config can be exported with `export config`
        """
        output = self.run_export_cmd("config")
        yml = yaml.load(output, Loader=yaml.FullLoader)
        assert isinstance(yml, dict)
