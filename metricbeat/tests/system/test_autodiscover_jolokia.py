import os
import metricbeat
import unittest
from nose.plugins.attrib import attr


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['jolokia']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_jolokia_jmx(self):
        """
        jolokia autodiscover with jmx metricset
        """

        self.render_config_template(
            autodiscover={
                'jolokia': {
                    'interfaces': '''
                      - name: any
                        interval: 120s
                    ''',
                    'templates': '''
                      - condition:
                          contains:
                            jolokia.server.product: "tomcat"
                        config:
                          - module: jolokia
                            metricsets: ["jmx"]
                            hosts: "${data.jolokia.url}"
                            namespace: test
                            jmx.mappings:
                            - mbean: "java.lang:type=GarbageCollector,name=PS MarkSweep"
                              attributes:
                              - attr: CollectionCount
                                field: gc.collection_count

                    ''',
                },
            },
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)
        evt = output[0]
        print(evt)

        assert evt["jolokia"]["test"]["gc"]["collection_count"] >= 0
