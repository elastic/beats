import os
import metricbeat
from nose.plugins.attrib import attr

ZK_FIELDS = metricbeat.COMMON_FIELDS + ["zookeeper-mntr"]

MNTR_FIELDS = ["zk_version", "zk_avg_latency", "zk_max_latency",
               "zk_min_latency", "zk_packets_received", "zk_packets_sent",
               "zk_outstanding_requests", "zk_server_state", "zk_znode_count",
               "zk_watch_count", "zk_ephemerals_count",
               "zk_approximate_data_size", "zk_followers",
               "zk_synced_followers", "zk_pending_syncs",
               "zk_open_file_descriptor_count", "zk_max_file_descriptor_count",
               "zk_num_alive_connections"]


class ZooKeeperMntrTest(metricbeat.BaseTest):
    @attr('integration')
    def test_output(self):
        """
        ZooKeeper mntr module outputs an event.
        """
        self.render_config_template(modules=[{
            "name": "zookeeper",
            "metricsets": ["mntr"],
            "hosts": [os.getenv('ZOOKEEPER_HOST') + ':' + os.getenv(
                'ZOOKEEPER_PORT')],
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log, "ERR|WARN")

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertItemsEqual(ZK_FIELDS, evt.keys())
        zk_mntr = evt["zookeeper-mntr"]
        self.assertItemsEqual(MNTR_FIELDS, zk_mntr.keys())

        self.assert_fields_are_documented(evt)
