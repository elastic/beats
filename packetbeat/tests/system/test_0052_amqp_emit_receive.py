from packetbeat import BaseTest


class Test(BaseTest):

    def test_amqp_emit_receive(self):
        self.render_config_template(
            amqp_ports=[5672],
        )
        self.run_packetbeat(pcap="amqp_emit_receive.pcap",
                            debug_selectors=["amqp,tcp,publish"])

        objs = self.read_output()
        assert all([o["type"] == "amqp" for o in objs])
        assert len(objs) == 7
        assert all([o["port"] == 5672 for o in objs])

        assert objs[0]["method"] == "exchange.declare"
        assert objs[0]["status"] == "OK"
        assert objs[0]["amqp.exchange"] == "logs"
        assert objs[0]["amqp.durable"] == True
        assert objs[0]["amqp.exchange-type"] == "fanout"
        assert objs[0]["amqp.passive"] == False
        assert objs[0]["amqp.no-wait"] == False

        assert objs[1]["method"] == "queue.declare"
        assert objs[1]["status"] == "OK"
        assert objs[1]["amqp.queue"] != ""
        assert objs[1]["amqp.exclusive"] == True
        assert objs[1]["amqp.no-wait"] == False
        assert objs[1]["amqp.durable"] == False
        assert objs[1]["amqp.auto-delete"] == False
        assert objs[1]["amqp.passive"] == False

        assert objs[2]["method"] == "queue.bind"
        assert objs[2]["status"] == "OK"
        assert objs[2]["amqp.queue"] != ""
        assert objs[2]["amqp.exchange"] == "logs"
        assert objs[2]["amqp.no-wait"] == False

        assert objs[3]["method"] == "basic.consume"
        assert objs[3]["status"] == "OK"
        assert objs[3]["amqp.queue"] != ""
        assert objs[3]["amqp.no-ack"] == True
        assert objs[3]["amqp.no-wait"] == False
        assert objs[3]["amqp.no-local"] == False
        assert objs[3]["amqp.exclusive"] == False

        assert objs[4]["method"] == "exchange.declare"
        assert objs[4]["status"] == "OK"
        assert objs[4]["amqp.exchange"] == "logs"
        assert objs[4]["amqp.durable"] == True
        assert objs[4]["amqp.exchange-type"] == "fanout"
        assert objs[4]["amqp.passive"] == False
        assert objs[4]["amqp.no-wait"] == False

        assert objs[5]["method"] == "basic.publish"
        assert objs[5]["status"] == "OK"
        assert objs[5]["amqp.content-type"] == "text/plain"
        assert objs[5]["amqp.exchange"] == "logs"
        assert objs[5]["amqp.immediate"] == False
        assert objs[5]["amqp.mandatory"] == False

        assert objs[6]["method"] == "basic.deliver"
        assert objs[6]["status"] == "OK"
        assert objs[6]["amqp.content-type"] == "text/plain"
        assert objs[6]["amqp.delivery-tag"] == 1
        assert objs[6]["amqp.exchange"] == "logs"
        assert objs[6]["amqp.redelivered"] == False
