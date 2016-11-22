from base import BaseTest
from nose.plugins.attrib import attr

import os
import subprocess
import unittest
import re
from nose.plugins.skip import Skip, SkipTest


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

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_export_dashboard(self):
        """
        Test export dashboards and remove unsupported characters
        """

        raise SkipTest
        # This test is currently skipped as it does not work.
        # The test fails as soon as there are dashboards loaded which can happen if
        # test_load_dashboard was run previously. Also this test should pass exactly
        # when dashboards exist, means it should first load some "wrong" dashboards
        # In addition, this test should not write to the beats directory but to a
        # temporary directory and check the files there.

        beats = ["metricbeat", "packetbeat", "filebeat", "winlogbeat"]

        for beat in beats:
            if os.name == "nt":
                path = "..\..\..\\"+ beat + "\etc\kibana"
            else:
                path = "../../../"+ beat + "/etc/kibana"

            command = "python ../../../dev-tools/export_dashboards.py --url http://"+ self.get_elasticsearch_host() + " --dir " + path + " --regex " + beat + "-*"

            if os.name == "nt":
                command = "python ..\..\..\dev-tools/export_dashboards.py --url http://"+ self.get_elasticsearch_host() + " --dir " + path + " --regex " + beat + "-*"

            print command

            p = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            content, err = p.communicate()

            assert p.returncode == 0

            files = os.listdir(path)

            for f in files:
                self.assertIsNone(re.search('[:\>\<"/\\\|\?\*]', f))


    def get_elasticsearch_host(self):
        return os.getenv('ES_HOST', 'localhost') + ':' + os.getenv('ES_PORT', '9200')
