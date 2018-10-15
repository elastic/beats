import os
import metricbeat
import unittest
from nose.plugins.attrib import attr
from nose.plugins.skip import SkipTest


class KafkaTest(metricbeat.BaseTest):
    COMPOSE_SERVICES = ['kafka']
    VERSION = "2.0.0"

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_partition(self):
        """
        kafka partition metricset test
        """

        self.create_topic()

        self.render_config_template(modules=[{
            "name": "kafka",
            "metricsets": ["partition"],
            "hosts": self.get_hosts(),
            "period": "1s",
            "version": self.VERSION,
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)
        evt = output[0]
        print(evt)

        self.assert_fields_are_documented(evt)

    def create_topic(self):
        from kafka import KafkaProducer

        producer = KafkaProducer(bootstrap_servers=self.get_hosts()[0],
                                 retries=20, retry_backoff_ms=500)
        producer.send('foobar', b'some_message_bytes')

    def get_hosts(self):
        return [self.compose_hosts()[0] + ':' +
                os.getenv('KAFKA_PORT', '9092')]


class Kafka_1_1_0_Test(KafkaTest):
    COMPOSE_SERVICES = ['kafka_1_1_0']
    VERSION = "1.1.0"


class Kafka_0_10_2_Test(KafkaTest):
    COMPOSE_SERVICES = ['kafka_0_10_2']
    VERSION = "0.10.2"
