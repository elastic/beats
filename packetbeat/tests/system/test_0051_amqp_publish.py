from packetbeat import BaseTest


class Test(BaseTest):

    def test_amqp_publish(self):
        self.render_config_template(
            amqp_ports=[5672],
            amqp_send_request=True
        )
        self.run_packetbeat(pcap="amqp_publish.pcap",
                            debug_selectors=["amqp,tcp,publish"])

        objs = self.read_output()
        assert all([o["type"] == "amqp" for o in objs])
        assert len(objs) == 2
        assert all([o["server.port"] == 5672 for o in objs])

        assert objs[0]["method"] == "queue.declare"
        assert objs[0]["status"] == "OK"
        assert objs[0]["amqp.queue"] == "hello"
        assert objs[0]["amqp.durable"] == False
        assert objs[0]["amqp.auto-delete"] == False
        assert objs[0]["amqp.exclusive"] == False
        assert objs[0]["amqp.no-wait"] == False

        assert objs[1]["method"] == "basic.publish"
        assert objs[1]["status"] == "OK"
        assert objs[1]["request"] == "hello"
        assert objs[1]["amqp.routing-key"] == "hello"
        assert objs[1]["amqp.mandatory"] == False
        assert objs[1]["amqp.immediate"] == False
        assert objs[1]["amqp.content-type"] == "text/plain"
