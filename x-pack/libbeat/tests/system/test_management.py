import sys
import os
import glob
import json
import requests
import string
import random
import unittest
import time
from elasticsearch import Elasticsearch
from os import path


from base import BaseTest


# Disable because waiting artifacts from https://github.com/elastic/kibana/pull/31660
INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)
# INTEGRATION_TESTS = False
TIMEOUT = 2 * 60


class TestManagement(BaseTest):

    def setUp(self):
        super(TestManagement, self).setUp()
        # NOTES: Theses options are linked to the specific of the docker compose environment for
        # CM.
        self.es_host = os.getenv('ES_HOST', 'localhost') + ":" + os.getenv('ES_POST', '9200')
        self.es_user = "myelastic"
        self.es_pass = "changeme"
        self.es = Elasticsearch([self.get_elasticsearch_url()], verify_certs=True)
        self.keystore_path = self.working_dir + "/data/keystore"

        if path.exists(self.keystore_path):
            os.Remove(self.keystore_path)

    @unittest.skipIf(not INTEGRATION_TESTS,
                     "integration tests are disabled, run with INTEGRATION_TESTS=1 to enable them.")
    def test_enroll(self):
        """
        Enroll the beat in Kibana Central Management
        """

        assert len(glob.glob(os.path.join(self.working_dir, "mockbeat.yml.*.bak"))) == 0

        # We don't care about this as it will be replaced by enrollment
        # process:
        config_path = os.path.join(self.working_dir, "mockbeat.yml")
        self.render_config_template("mockbeat", config_path, keystore_path=self.keystore_path)

        config_content = open(config_path, 'r').read()

        exit_code = self.enroll(self.es_user, self.es_pass)

        assert exit_code == 0

        assert self.log_contains("Enrolled and ready to retrieve settings")

        # Enroll creates a keystore (to store access token)
        assert os.path.isfile(os.path.join(
            self.working_dir, "data/keystore"))

        # New settings file is in place now
        new_content = open(config_path, 'r').read()
        assert config_content != new_content

        # Settings backup has been created
        backup_file = glob.glob(os.path.join(self.working_dir, "mockbeat.yml.*.bak"))[0]
        assert os.path.isfile(backup_file)
        backup_content = open(backup_file).read()
        assert config_content == backup_content

    @unittest.skipIf(not INTEGRATION_TESTS,
                     "integration tests are disabled, run with INTEGRATION_TESTS=1 to enable them.")
    def test_enroll_bad_pw(self):
        """
        Try to enroll the beat in Kibana Central Management with a bad password
        """
        # We don't care about this as it will be replaced by enrollment
        # process:
        config_path = os.path.join(self.working_dir, "mockbeat.yml")
        self.render_config_template("mockbeat", config_path, keystore_path=self.keystore_path)

        config_content = open(config_path, 'r').read()

        exit_code = self.enroll("not", 'wrong password')

        assert exit_code == 1

        # Keystore wasn't created
        assert not os.path.isfile(os.path.join(
            self.working_dir, "data/keystore"))

        # Settings hasn't changed
        new_content = open(config_path, 'r').read()
        assert config_content == new_content

    @unittest.skipIf(not INTEGRATION_TESTS,
                     "integration tests are disabled, run with INTEGRATION_TESTS=1 to enable them.")
    def test_fetch_configs(self):
        """
        Config is retrieved from Central Management and updates are applied
        """
        # Enroll the beat
        config_path = os.path.join(self.working_dir, "mockbeat.yml")
        self.render_config_template("mockbeat", config_path, keystore_path=self.keystore_path)
        exit_code = self.enroll(self.es_user, self.es_pass)
        assert exit_code == 0

        index = self.random_index()
        # Configure an output
        self.create_and_assing_tag([
            {
                "type": "output",
                "config": {
                        "_sub_type": "elasticsearch",
                        "hosts": [self.es_host],
                        "username": self.es_user,
                        "password": self.es_pass,
                        "index": index,
                },
                "id": "myconfig",
            }
        ])

        # Start beat
        proc = self.start_beat(extra_args=[
            "-E", "management.period=1s",
            "-E", "keystore.path=%s" % self.keystore_path,
        ])

        # Wait for beat to apply new conf
        self.wait_log_contains("Applying settings for output")

        self.wait_until(lambda: self.log_contains("PublishEvents: "), max_timeout=TIMEOUT)

        self.wait_documents(index, 1)

        index2 = self.random_index()

        # Update output configuration
        self.create_and_assing_tag([
            {
                "type": "output",
                "config": {
                        "_sub_type": "elasticsearch",
                        "hosts": [self.es_host],
                        "username": self.es_user,
                        "password": self.es_pass,
                        "index": index2,
                },
                "id": "myconfig",
            }
        ])
        self.wait_log_contains("Applying settings for output")
        self.wait_until(lambda: self.log_contains("PublishEvents: "), max_timeout=TIMEOUT)
        self.wait_documents(index2, 1)

        proc.check_kill_and_wait()

    @unittest.skipIf(not INTEGRATION_TESTS,
                     "integration tests are disabled, run with INTEGRATION_TESTS=1 to enable them.")
    def test_configs_cache(self):
        """
        Config cache is used if Kibana is not available
        """
        # Enroll the beat
        config_path = os.path.join(self.working_dir, "mockbeat.yml")
        self.render_config_template("mockbeat", config_path, keystore_path=self.keystore_path)
        exit_code = self.enroll(self.es_user, self.es_pass)
        assert exit_code == 0

        index = self.random_index()

        # Update output configuration
        self.create_and_assing_tag([
            {
                "type": "output",
                "config": {
                        "_sub_type": "elasticsearch",
                        "hosts": [self.es_host],
                        "username": self.es_user,
                        "password": self.es_pass,
                        "index": index,
                }
            }
        ])

        # Start beat
        proc = self.start_beat(extra_args=[
            "-E", "management.period=1s",
            "-E", "keystore.path=%s" % self.keystore_path,
        ])

        self.wait_until(lambda: self.log_contains("PublishEvents: "), max_timeout=TIMEOUT)
        self.wait_documents(index, 1)
        proc.check_kill_and_wait()

        # Remove the index
        self.es.indices.delete(index)

        # Cache should exists already, start with wrong kibana settings:
        proc = self.start_beat(extra_args=[
            "-E", "management.period=1s",
            "-E", "management.kibana.host=wronghost",
            "-E", "management.kibana.timeout=0.5s",
            "-E", "keystore.path=%s" % self.keystore_path,
        ])

        self.wait_until(lambda: self.log_contains("PublishEvents: "), max_timeout=TIMEOUT)
        self.wait_documents(index, 1)
        proc.check_kill_and_wait()

    def enroll(self, user, password):
        return self.run_beat(
            extra_args=["enroll", self.get_kibana_url(),
                        "--password", "env:PASS", "--username", user, "--force"],
            logging_args=["-v", "-d", "*"],
            env={
                'PASS': password,
            })

    def check_kibana_status(self):
        headers = {
            "kbn-xsrf": "1"
        }

        # Create tag
        url = self.get_kibana_url() + "/api/status"

        r = requests.get(url, headers=headers,
                         auth=(self.es_user, self.es_pass))

    def create_and_assing_tag(self, blocks):
        tag_name = "test%d" % int(time.time() * 1000)
        headers = {
            "kbn-xsrf": "1"
        }

        # Create tag
        url = self.get_kibana_url() + "/api/beats/tag/" + tag_name
        data = {
            "color": "#DD0A73",
            "name": tag_name,
        }

        r = requests.put(url, json=data, headers=headers,
                         auth=(self.es_user, self.es_pass))
        assert r.status_code in (200, 201)

        # Create blocks
        url = self.get_kibana_url() + "/api/beats/configurations"
        for b in blocks:
            b["tag"] = tag_name

        r = requests.put(url, json=blocks, headers=headers,
                         auth=(self.es_user, self.es_pass))
        assert r.status_code in (200, 201)

        # Retrieve beat ID
        meta = json.loads(
            open(os.path.join(self.working_dir, 'data', 'meta.json'), 'r').read())

        # Assign tag to beat
        data = {"assignments": [{"beatId": meta["uuid"], "tag": tag_name}]}
        url = self.get_kibana_url() + "/api/beats/agents_tags/assignments"
        r = requests.post(url, json=data, headers=headers,
                          auth=(self.es_user, self.es_pass))

        assert r.status_code == 200

    def get_elasticsearch_url(self):
        return 'http://' + self.es_user + ":" + self.es_pass + '@' + os.getenv('ES_HOST', 'localhost') + ':' + os.getenv('ES_PORT', '5601')

    def get_kibana_url(self):
        return 'http://' + os.getenv('KIBANA_HOST', 'kibana') + ':' + os.getenv('KIBANA_PORT', '5601')

    def random_index(self):
        return ''.join(random.choice(string.ascii_lowercase) for i in range(10))

    def check_document_count(self, index, count):
        try:
            self.es.indices.refresh(index=index)
            return self.es.search(index=index, body={"query": {"match_all": {}, "allow_partial_search_results": "true"}})['hits']['total'] >= count
        except:
            return False

    def wait_documents(self, index, count):
        self.wait_until(lambda: self.check_document_count(index, count), max_timeout=TIMEOUT, poll_interval=1)
