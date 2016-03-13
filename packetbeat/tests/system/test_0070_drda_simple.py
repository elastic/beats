from packetbeat import BaseTest

def check_fields(flow, fields):
    for k, v in fields.iteritems():
        assert flow[k] == v

class Test(BaseTest):
    def test_simple(self):
        self.render_config_template(
            drda_ports=[1527],
        )

        self.run_packetbeat(pcap="drda/drda_simple.pcap", debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 21
        check_fields(objs[8], {
            'query': 'select * from derbyDb',
            'bytes_in': 258,
            'bytes_out': 511,
        })
