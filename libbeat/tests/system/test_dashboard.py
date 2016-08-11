from base import BaseTest

import os
import subprocess


class Test(BaseTest):
    def test_load_dashboard(self):
        """
        Test loading dashboards for all beats
        """
        beats = ["metricbeat", "packetbeat", "filebeat", "winlogbeat"]

        for beat in beats:
            command = "../../../dev-tools/import_dashboards.sh -l http://"+ self.get_elasticsearch_host() + " -dir ../../../"+ beat + "/etc/kibana"

            if os.name == "nt":
                command = "..\..\..\dev-tools\import_dashboards.ps1 -dir ..\..\..\\" + beat + "\etc\kibana"

            p = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            content, err = p.communicate()

            assert p.returncode == 0

    def get_elasticsearch_host(self):
        return os.getenv('ES_HOST', 'localhost') + ':' + os.getenv('ES_PORT', '9200')
