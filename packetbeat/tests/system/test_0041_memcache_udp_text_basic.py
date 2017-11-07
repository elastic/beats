from packetbeat import BaseTest

import six

# import pprint
#
# pp = pprint.PrettyPrinter()
#
#
# def pretty(*k, **kw):
#     pp.pprint(*k, **kw)


class Test(BaseTest):

    def _run(self, pcap):
        self.render_config_template(
            memcache_udp_transaction_timeout=10
        )
        self.run_packetbeat(pcap=pcap,
                            debug_selectors=["memcache", "udp", "publish"])
        objs = self.read_output()
        self.assert_common(objs)
        return objs

    def assert_common(self, objs):
        # check client ip are not mixed up
        assert all(o['client_ip'] == '192.168.188.37' for o in objs)
        assert all(o['ip'] == '192.168.188.38' for o in objs)
        assert all(o['port'] == 11211 for o in objs)

        # check transport layer always tcp
        assert all(o['type'] == 'memcache' for o in objs)
        assert all(o['transport'] == 'udp' for o in objs)
        assert all(o['memcache.protocol_type'] == 'text' for o in objs)

    def test_store(self):
        objs = self._run("memcache/memcache_text_udp_single_store.pcap")

        # all transactions succeed
        assert all(o['status'] == 'OK' for o in objs)
        assert len(objs) == 1

        set = objs[0]
        assert set['memcache.request.command'] == 'set'
        assert set['memcache.request.type'] == 'Store'
        assert set['memcache.request.keys'] == ['test_key']
        assert set['memcache.request.exptime'] == 0
        assert set['memcache.request.bytes'] == 1024
        assert set['memcache.request.count_values'] == 1
        assert set['memcache.request.noreply']

    def test_multi_store(self):
        objs = self._run("memcache/memcache_text_udp_multi_store.pcap")

        # all transactions succeed
        assert all(o['status'] == 'OK' for o in objs)
        assert len(objs) == 3

        sets = dict([(o['memcache.request.keys'][0], o) for o in objs[0:3]])
        assert sorted(sets.keys()) == ['k1', 'k2', 'k3']
        assert sets['k1']['memcache.request.bytes'] == 100
        assert sets['k2']['memcache.request.bytes'] == 20
        assert sets['k3']['memcache.request.bytes'] == 10
        assert all(o['memcache.request.noreply'] for o in six.itervalues(sets))

    def test_delete(self):
        objs = self._run('memcache/memcache_text_udp_delete.pcap')

        # all transactions succeed
        assert all(o['status'] == 'OK' for o in objs)

        assert len(objs) == 2
        delete, set = sorted(objs, key=lambda x: x['memcache.request.command'])

        # check set command
        assert set['memcache.request.command'] == 'set'
        assert set['memcache.request.type'] == 'Store'
        assert set['memcache.request.keys'] == ['key']
        assert set['memcache.request.noreply']

        # check delete command
        assert delete['memcache.request.command'] == 'delete'
        assert delete['memcache.request.type'] == 'Delete'
        assert delete['memcache.request.keys'] == ['key']
        assert delete['memcache.request.noreply']

    def test_counter_ops(self):
        objs = self._run('memcache/memcache_text_udp_counter_ops.pcap')

        # all transactions succeed
        assert all(o['status'] == 'OK' for o in objs)

        assert len(objs) == 3
        dec, inc, set = sorted(objs,
                               key=lambda x: x['memcache.request.command'])

        # check set command
        assert set['memcache.request.command'] == 'set'
        assert set['memcache.request.type'] == 'Store'
        assert set['memcache.request.keys'] == ['cnt']
        assert set['memcache.request.noreply']

        assert inc['memcache.request.command'] == 'incr'
        assert inc['memcache.request.delta'] == 2
        assert inc['memcache.request.keys'] == ['cnt']
        assert inc['memcache.request.noreply']

        assert dec['memcache.request.command'] == 'decr'
        assert dec['memcache.request.delta'] == 5
        assert dec['memcache.request.keys'] == ['cnt']
        assert dec['memcache.request.noreply']
