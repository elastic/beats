import os

from heartbeat import BaseTest


class Test(BaseTest):

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
        self.assertDictContainsSubset(expected, doc)
        return doc
