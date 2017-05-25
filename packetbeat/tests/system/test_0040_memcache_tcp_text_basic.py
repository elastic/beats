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
        self.render_config_template()
        self.run_packetbeat(pcap=pcap,
                            debug_selectors=["memcache", "tcp", "publish"])
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
        assert all(o['transport'] == 'tcp' for o in objs)
        assert all(o['memcache.protocol_type'] == 'text' for o in objs)

    def test_store_load(self):
        objs = self._run("memcache/memcache_text_tcp_single_load_store.pcap")

        # all transactions succeed
        assert all(o['status'] == 'OK' for o in objs)

        # validate set transaction
        set = objs[0]
        assert set['memcache.request.command'] == 'set'
        assert set['memcache.request.type'] == 'Store'
        assert set['memcache.request.keys'] == ['test_key']
        assert set['memcache.request.exptime'] == 0
        assert set['memcache.request.bytes'] == 2046
        assert set['memcache.request.count_values'] == 1
        assert not set['memcache.request.noreply']
        assert set['memcache.response.type'] == 'Success'

        # validate get transaction
        get = objs[1]
        assert get['memcache.request.command'] == 'get'
        assert get['memcache.request.keys'] == ['test_key']
        assert get['memcache.request.type'] == 'Load'
        assert get['memcache.response.type'] == 'Load'
        assert get['memcache.response.count_values'] == 1
        assert get['memcache.response.keys'] == ['test_key']

    def test_multi_store_load(self):
        objs = self._run("memcache/memcache_text_tcp_multi_store_load.pcap")

        # all transactions succeed
        assert all(o['status'] == 'OK' for o in objs)

        sets = dict([(o['memcache.request.keys'][0], o) for o in objs[0:3]])
        assert sorted(sets.keys()) == ['k1', 'k2', 'k3']
        assert sets['k1']['memcache.request.bytes'] == 100
        assert sets['k2']['memcache.request.bytes'] == 20
        assert sets['k3']['memcache.request.bytes'] == 10
        assert all(o['memcache.response.type'] == 'Success' for o in six.itervalues(sets))

        get = objs[3]
        assert get['memcache.request.keys'] == ['x', 'k1', 'k2', 'k3', 'y']
        assert get['memcache.response.keys'] == ['k1', 'k2', 'k3']
        assert get['memcache.response.bytes'] == 130
        assert get['memcache.response.count_values'] == 3

    def test_counter_ops(self):
        objs = self._run('memcache/memcache_text_tcp_counter_ops.pcap')

        # all transactions succeed
        assert all(o['status'] == 'OK' for o in objs)

        set, inc, dec, get, _ = objs

        # check set command
        assert set['memcache.request.command'] == 'set'
        assert set['memcache.request.type'] == 'Store'
        assert set['memcache.request.keys'] == ['cnt']

        assert inc['memcache.request.command'] == 'incr'
        assert inc['memcache.request.delta'] == 2
        assert inc['memcache.request.keys'] == ['cnt']
        assert inc['memcache.response.value'] == 2

        assert dec['memcache.request.command'] == 'decr'
        assert dec['memcache.request.delta'] == 5
        assert dec['memcache.request.keys'] == ['cnt']
        assert dec['memcache.response.value'] == 0

        assert get['memcache.request.command'] == 'get'
        assert get['memcache.request.type'] == 'Load'
        assert get['memcache.request.keys'] == ['cnt']

    def test_delete(self):
        objs = self._run('memcache/memcache_text_tcp_delete.pcap')

        # all transactions succeed
        assert all(o['status'] == 'OK' for o in objs)

        set, delete, get, _ = objs

        # check set command
        assert set['memcache.request.command'] == 'set'
        assert set['memcache.request.type'] == 'Store'
        assert set['memcache.request.keys'] == ['key']

        # check delete command
        assert delete['memcache.request.command'] == 'delete'
        assert delete['memcache.request.type'] == 'Delete'
        assert delete['memcache.request.keys'] == ['key']
        assert delete['memcache.response.type'] == 'Success'

        # check get command on deleted key
        assert get['memcache.request.command'] == 'get'
        assert get['memcache.request.type'] == 'Load'
        assert get['memcache.request.keys'] == ['key']
        assert get['memcache.response.command'] == 'END'  # no keys

    def test_stats(self):
        objs = self._run('memcache/memcache_text_tcp_stats.pcap')

        # all transactions succeed
        assert all(o['status'] == 'OK' for o in objs)

        trans = objs[2]  # get stats command

        assert trans['memcache.request.command'] == 'stats'
        assert 'memcache.request.raw_args' in trans
        assert trans['memcache.request.type'] == 'Stats'
        assert trans['memcache.response.command'] == 'STAT'
        assert trans['memcache.response.type'] == 'Stats'

        # check all fields are set
        entries = trans['memcache.response.stats']
        assert all(e['name'] and e['value'] for e in entries)
