from base import BaseTest
from nose.plugins.attrib import attr

import os
import subprocess
import unittest

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)

class Test(BaseTest):

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_load_dashboard(self):
        """
        Test loading dashboards for all beats
        """
        beats = ["metricbeat", "packetbeat", "filebeat", "winlogbeat"]

        for beat in beats:
            command = "go run ../../dashboards/import_dashboards.go -es http://"+ self.get_elasticsearch_host() + " -dir ../../../"+ beat + "/etc/kibana"

            if os.name == "nt":
                command = "go run ..\..\dashboards\import_dashboards.go -es http:\\"+self.get_elasticsearch_host() + " -dir ..\..\..\\" + beat + "\etc\kibana"

            print command
            p = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            content, err = p.communicate()

            assert p.returncode == 0

    def get_elasticsearch_host(self):
        return os.getenv('ES_HOST', 'localhost') + ':' + os.getenv('ES_PORT', '9200')
