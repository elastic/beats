import os
from os import path

from base import BaseTest
from keystore import KeystoreBase


class TestKeystore(KeystoreBase):
    """
    Test Keystore variable replacement
    """

    def setUp(self):
        super(BaseTest, self).setUp()
        self.keystore_path = self.working_dir + "/data/keystore"

        if path.exists(self.keystore_path):
            os.Remove(self.keystore_path)

    def test_keystore_with_present_key(self):
        """
        Test that we correctly to string replacement with values from the keystore
        """

        key = "mysecretpath"
        secret = path.join(self.working_dir, "thisisultrasecretpath")

        self.render_config_template("mockbeat",
                                    keystore_path=self.keystore_path,
                                    output_file_path="${%s}" % key)

        exit_code = self.run_beat(extra_args=["keystore", "create"],
                                  config="mockbeat.yml")
        assert exit_code == 0

        self.add_secret(key, secret)
        proc = self.start_beat(config="mockbeat.yml")
        self.wait_until(lambda: self.log_contains("ackloop:  done send ack"))
        proc.check_kill_and_wait()
        assert path.exists(secret)

    def test_keystore_with_key_not_present(self):
        key = "elasticsearch_host"

        self.render_config_template(keystore_path=self.keystore_path, elasticsearch={
            'hosts': "${%s}:9200" % key
        })

        exit_code = self.run_beat()
        assert self.log_contains(
            "missing field accessing 'output.elasticsearch.hosts'")
        assert exit_code == 1

    def test_keystore_with_nested_key(self):
        """
        test that we support nested key
        """

        key = "output.elasticsearch.hosts.0"
        secret = path.join(self.working_dir, "myeleasticsearchsecrethost")

        self.render_config_template("mockbeat",
                                    keystore_path=self.keystore_path,
                                    output_file_path="${%s}" % key)

        exit_code = self.run_beat(extra_args=["keystore", "create"],
                                  config="mockbeat.yml")
        assert exit_code == 0

        self.add_secret(key, secret)
        proc = self.start_beat(config="mockbeat.yml")
        self.wait_until(lambda: self.log_contains("ackloop:  done send ack"))
        proc.check_kill_and_wait()
        assert path.exists(secret)

    def test_export_config_with_keystore(self):
        """
        Test export config works and doesn't expose keystore value
        """
        key = "asecret"
        secret = "asecretvalue"

        self.render_config_template(keystore_path=self.keystore_path, elasticsearch={
            'hosts': "${%s}" % key
        })

        exit_code = self.run_beat(extra_args=["keystore", "create"])
        assert exit_code == 0

        self.add_secret(key, value=secret)
        exit_code = self.run_beat(extra_args=["export", "config"])

        assert exit_code == 0
        assert self.log_contains(secret) == False
        assert self.log_contains("${%s}" % key)
