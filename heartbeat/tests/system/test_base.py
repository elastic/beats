import os
import unittest

from heartbeat import BaseTest
from beat.beat import INTEGRATION_TESTS
from beat import common_tests
from time import sleep


class Test(BaseTest, common_tests.TestExportsMixin):

    def test_base(self):
        """
        Basic test with exiting Heartbeat normally
        """

        config = {
            "monitors": [
                {
                    "type": "http",
                    "urls": ["http://localhost:9200"],
                }
            ]
        }

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            **config
        )

        heartbeat_proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("heartbeat is running"))
        heartbeat_proc.check_kill_and_wait()

    def test_run_once(self):
        """
        Basic test with exiting Heartbeat normally
        """

        config = {
            "run_once": True,
            "monitors": [
                {
                    "type": "http",
                    "id": "http-check",
                    "urls": ["http://localhost:9200"],
                },
                {
                    "type": "tcp",
                    "id": "tcp-check",
                    "hosts": ["localhost:9200"],
                }
            ]
        }

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            **config
        )

        heartbeat_proc = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=2))
        self.wait_until(lambda: self.log_contains("Ending run_once run"))
        heartbeat_proc.check_wait()

    def test_disabled(self):
        """
        Basic test against a disabled monitor
        """

        config = {
            "monitors": [
                {
                    "type": "http",
                    "enabled": "false",
                    "urls": ["http://localhost:9200"],
                }
            ]
        }

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            **config
        )

        heartbeat_proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("heartbeat is running"))
        heartbeat_proc.check_kill_and_wait()

    def test_fields_under_root(self):
        """
        Basic test with fields and tags in monitor
        """

        self.run_fields(
            local={
                "tags": ["local"],
                "fields_under_root": True,
                "fields": {"local": "field", "env": "dev"},
            },
            top={
                "tags": ["global"],
                "fields": {
                    "global": "field",
                    "env": "prod",
                    "level": "overwrite"
                },
                "fields_under_root": True,
            },
            expected={
                "tags": ["global", "local"],
                "global": "field",
                "local": "field",
                "env": "dev"
            }
        )

    def test_fields_not_under_root(self):
        """
        Basic test with fields and tags (not under root)
        """
        self.run_fields(
            local={
                "tags": ["local"],
                "fields": {"local": "field", "env": "dev", "num": 1}
            },
            top={
                "tags": ["global"],
                "fields": {
                    "global": "field",
                    "env": "prod",
                    "level": "overwrite",
                    "num": 0
                }
            },
            expected={
                "tags": ["global", "local"],
                "fields.global": "field",
                "fields.local": "field",
                "fields.env": "dev"
            }
        )

    def test_host_fields_not_present(self):
        """
        Ensure that libbeat isn't adding any host.* fields
        """
        monitor = {
            "type": "http",
            "urls": ["http://localhost:9200"],
        }
        config = {
            "monitors": [monitor]
        }

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/*",
            **config
        )

        heartbeat_proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        heartbeat_proc.check_kill_and_wait()
        doc = self.read_output()[0]

        assert "host.name" not in doc

    def run_fields(self, expected, local=None, top=None):
        monitor = {
            "type": "http",
            "urls": ["http://localhost:9200"],
        }
        if local:
            monitor.update(local)

        config = {
            "monitors": [monitor]
        }
        if top:
            config.update(top)

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/*",
            **config
        )

        heartbeat_proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        heartbeat_proc.check_kill_and_wait()

        doc = self.read_output()[0]
        assert expected.items() <= doc.items()
        return doc

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    def test_index_management(self):
        """
        Test that the template can be loaded with `setup --index-management`
        """
        es = self.get_elasticsearch_instance()
        self.render_config_template(
            monitors=[{
                "type": "http",
                "urls": ["http://localhost:9200"],
            }],
            elasticsearch=self.get_elasticsearch_template_config()
        )
        exit_code = self.run_beat(extra_args=["setup", "--index-management"])

        assert exit_code == 0
        assert self.log_contains('Loaded index template')
        assert len(es.cat.templates(name='heartbeat-*', h='name')) > 0

    def test_dataset(self):
        """
        Test that event.dataset is set to `uptime`
        """
        self.render_config_template(
            monitors=[
                {
                    "type": "http",
                    "urls": ["http://localhost:9200"]
                },
                {
                    "type": "tcp",
                    "hosts": ["localhost:9200"]
                }
            ]
        )

        try:
            heartbeat_proc = self.start_beat()
            self.wait_until(lambda: self.output_lines() >= 2)
        finally:
            heartbeat_proc.check_kill_and_wait()

        for output in self.read_output():
            self.assertEqual(
                output["event.dataset"],
                output["monitor.type"],
                "Check for event.dataset in {} failed".format(output)
            )
