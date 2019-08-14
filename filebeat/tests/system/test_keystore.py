import os
from os import path

from filebeat import BaseTest
from beat.beat import Proc


class TestKeystore(BaseTest):
    """
    Test Keystore variable replacement
    """

    def setUp(self):
        super(BaseTest, self).setUp()
        self.keystore_path = self.working_dir + "/data/keystore"

    def test_keystore_with_present_key(self):
        """
        Test that we correctly do string replacement with values from the keystore
        """

        key = "elasticsearch_host"
        secret = "myeleasticsearchsecrethost"

        self.render_config_template(keystore_path=self.keystore_path, elasticsearch={
            'host': "${%s}:9200" % key
        })

        exit_code = self.run_beat(extra_args=["keystore", "create"])
        assert exit_code == 0

        self.add_secret(key, secret, True)
        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("myeleasticsearchsecrethost"))
        assert self.log_contains(secret)
        proc.kill_and_wait()

    def test_keystore_with_key_not_present(self):
        """
        Test that we return the template key when the key doesn't exist
        """
        key = "do_not_exist_elasticsearch_host"

        self.render_config_template(keystore_path=self.keystore_path, elasticsearch={
            'host': "${%s}:9200" % key
        })

        exit_code = self.run_beat()
        assert self.log_contains(
            "missing field accessing 'output.elasticsearch.hosts.0'")
        assert exit_code == 1

    def add_secret(self, key, value="hello world\n", force=False):
        """
        Add new secret using the --stdin option
        """
        args = [self.test_binary,
                "-systemTest",
                "-c", os.path.join(self.working_dir, self.beat_name + ".yml"),
                "-e", "-v", "-d", "*",
                "keystore", "add", key, "--stdin",
                ]

        if force:
            args.append("--force")

        proc = Proc(args, os.path.join(self.working_dir, self.beat_name + ".log"))

        os.write(proc.stdin_write, value)
        os.close(proc.stdin_write)

        return proc.start().wait()
