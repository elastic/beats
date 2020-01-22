from os import path
import os
import hashlib

from keystore import KeystoreBase


class TestCommandKeystore(KeystoreBase):
    """
    Test keystore subcommand

    """

    def setUp(self):
        super(TestCommandKeystore, self).setUp()

        self.keystore_path = self.working_dir + "/data/keystore"

        self.render_config_template(keystore_path=self.keystore_path)

        if path.exists(self.keystore_path):
            os.Remove(self.keystore_path)

    def test_keystore_create(self):
        """
        test keystore create command
        """
        exit_code = self.run_beat(extra_args=["keystore", "create"])

        assert exit_code == 0
        assert path.exists(self.keystore_path)

    def test_keystore_create_force(self):
        """
        Allow to override
        """

        self.run_beat(extra_args=["keystore", "create"])

        assert path.exists(self.keystore_path)
        digest_before = hashlib.sha256(
            open(self.keystore_path, 'rb').read()).digest()

        exit_code = self.run_beat(extra_args=["keystore", "create", "--force"])
        digest_after = hashlib.sha256(
            open(self.keystore_path, 'rb').read()).digest()

        assert exit_code == 0
        assert digest_before != digest_after

    def test_keystore_remove_no_key_no_keystore(self):
        """
        Remove a key that doesn't exist when the keystore doesn't exist
        """
        exit_code = self.run_beat(extra_args=["keystore", "remove", "mykey"])
        assert exit_code == 1
        assert "The filebeat keystore doesn't exist."

    def test_keystore_remove_non_existing_key(self):
        """
        Remove a key that doesn't exist in the keystore
        """
        self.run_beat(extra_args=["keystore", "create"])

        exit_code = self.run_beat(extra_args=["keystore", "remove", "mykey"])
        assert exit_code == 1

    def test_keystore_remove_existing_key(self):
        """
        Remove an key present in the keystore
        """
        self.run_beat(extra_args=["keystore", "create"])

        self.add_secret("mykey")
        exit_code = self.run_beat(extra_args=["keystore", "remove", "mykey"])

        assert exit_code == 0

    def test_keystore_remove_multiples_existing_keys(self):
        """
        Remove an key present in the keystore
        """

        self.run_beat(extra_args=["keystore", "create"])

        self.add_secret("willnotdelete")
        self.add_secret("myawesomekey")
        self.add_secret("mysuperkey")

        exit_code = self.run_beat(
            extra_args=["keystore", "remove", "mysuperkey", "myawesomekey"])

        assert exit_code == 0

        exit_code = self.run_beat(extra_args=["keystore", "list"])

        assert exit_code == 0

    def test_keystore_list(self):
        """
        list the available keys
        """

        self.run_beat(extra_args=["keystore", "create"])

        self.add_secret("willnotdelete")
        self.add_secret("myawesomekey")
        self.add_secret("mysuperkey")

        exit_code = self.run_beat(extra_args=["keystore", "list"])

        assert exit_code == 0

        assert self.log_contains("willnotdelete")
        assert self.log_contains("myawesomekey")
        assert self.log_contains("mysuperkey")

    def test_keystore_list_keys_on_an_empty_keystore(self):
        """
        List keys on an empty keystore should not return anything
        """
        exit_code = self.run_beat(extra_args=["keystore", "list"])
        assert exit_code == 0

    def test_keystore_add_secret_from_stdin(self):
        """
        Add a secret to the store using stdin
        """
        self.run_beat(extra_args=["keystore", "create"])
        exit_code = self.add_secret("willnotdelete")

        assert exit_code == 0

    def test_keystore_update_force(self):
        """
        Update an existing key using the --force flag
        """
        self.run_beat(extra_args=["keystore", "create"])

        self.add_secret("superkey")

        exit_code = self.add_secret("mysuperkey", "hello", True)

        assert exit_code == 0
