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

        key = "elasticsearch_host"
        secret = "myeleasticsearchsecrethost"

        self.render_config_template(keystore_path=self.keystore_path, elasticsearch={
            'hosts': "${%s}:9200" % key
        })

        exit_code = self.run_beat(extra_args=["keystore", "create"])
        assert exit_code == 0

        self.add_secret(key, secret)
        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("no such host"))
        assert self.log_contains(secret)
        proc.check_kill_and_wait()

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
        secret = "myeleasticsearchsecrethost"

        self.render_config_template(keystore_path=self.keystore_path, elasticsearch={
            'hosts': "${%s}" % key
        })

        exit_code = self.run_beat(extra_args=["keystore", "create"])
        assert exit_code == 0

        self.add_secret(key, secret)
        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("no such host"))
        assert self.log_contains(secret)
        proc.check_kill_and_wait()
