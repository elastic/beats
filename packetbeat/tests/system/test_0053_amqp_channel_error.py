from packetbeat import BaseTest


class Test(BaseTest):

    def test_amqp_channel_error(self):
        self.render_config_template(
            amqp_ports=[5672],
        )
        self.run_packetbeat(pcap="amqp_channel_error.pcap",
                            debug_selectors=["amqp,tcp,publish"])

        objs = self.read_output()
        assert all([o["type"] == "amqp" for o in objs])
        assert len(objs) == 3

        assert objs[0]["method"] == "exchange.declare"
        assert objs[0]["status"] == "OK"
        assert objs[0]["amqp.exchange"] == "titres"
        assert objs[0]["amqp.durable"] == True
        assert objs[0]["amqp.exchange-type"] == "fanout"
        assert objs[0]["amqp.passive"] == False
        assert objs[0]["amqp.no-wait"] == True

        assert objs[1]["method"] == "queue.declare"
        assert objs[1]["status"] == "OK"
        assert objs[1]["amqp.queue"] == "my_queue"
        assert objs[1]["amqp.exclusive"] == True
        assert objs[1]["amqp.no-wait"] == False
        assert objs[1]["amqp.durable"] == False
        assert objs[1]["amqp.auto-delete"] == False
        assert objs[1]["amqp.passive"] == False

        assert objs[2]["method"] == "channel.close"
        assert objs[2]["status"] == "Error"
        assert objs[2]["amqp.reply-code"] == 404
        assert objs[2]["amqp.reply-text"] == "NOT_FOUND - no exchange 'plop' in vhost '/'"
        assert objs[2]["amqp.class-id"] == 50
        assert objs[2]["amqp.method-id"] == 20
