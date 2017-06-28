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
        assert all(o['memcache.protocol_type'] == 'binary' for o in objs)

    def test_store_load(self):
        objs = self._run("memcache/memcache_bin_tcp_single_load_store.pcap")

        # all transactions succeed
        assert all(o['status'] == 'OK' for o in objs)

        # validate set transaction
        set = objs[0]
        assert set['memcache.request.opcode'] == 'Set'
        assert set['memcache.request.command'] == 'set'
        assert set['memcache.request.type'] == 'Store'
        assert set['memcache.request.keys'] == ['test_key']
        assert set['memcache.request.exptime'] == 0
        assert set['memcache.request.bytes'] == 2046
        assert set['memcache.request.count_values'] == 1
        assert set['memcache.response.opcode'] == 'Set'
        assert set['memcache.response.status'] == 'Success'
        assert 'memcache.response.cas_unique' in set
        assert (set['memcache.request.opaque'] ==
                set['memcache.response.opaque'])

        # validate get transaction
        get = objs[1]
        assert get['memcache.request.opcode'] == 'GetK'
        assert get['memcache.request.command'] == 'get'
        assert get['memcache.request.keys'] == ['test_key']
        assert get['memcache.request.type'] == 'Load'
        assert get['memcache.response.type'] == 'Load'
        assert get['memcache.response.count_values'] == 1
        assert get['memcache.response.bytes'] == 2046
        assert get['memcache.response.keys'] == ['test_key']
        assert 'memcache.response.cas_unique' in get
        assert (set['memcache.request.opaque'] ==
                set['memcache.response.opaque'])

    def test_multi_store_load(self):
        objs = self._run("memcache/memcache_bin_tcp_multi_store_load.pcap")

        # all transactions succeed
        assert all(o['status'] == 'OK' for o in objs)

        sets = dict([(o['memcache.request.keys'][0], o) for o in objs[0:3]])
        assert sorted(sets.keys()) == ['k1', 'k2', 'k3']
        assert sets['k1']['memcache.request.bytes'] == 100
        assert sets['k2']['memcache.request.bytes'] == 20
        assert sets['k3']['memcache.request.bytes'] == 10
        assert all(o['memcache.request.opcode'] == 'Set' for o in six.itervalues(sets))
        assert all('memcache.response.cas_unique' in o for o in six.itervalues(sets))
        assert all(o['memcache.response.status'] == 'Success' for o in six.itervalues(sets))
        assert all((o['memcache.request.opaque'] ==
                    o['memcache.response.opaque']) for o in six.itervalues(sets))

        gets = dict([(o['memcache.request.keys'][0], o) for o in objs[3:8]])
        # check all gets are quiet (transaction piping)
        assert all(o['memcache.request.opcode'] == 'GetKQ' for o in six.itervalues(gets))
        assert all(o['memcache.request.command'] == 'get' for o in six.itervalues(gets))
        assert all(o['memcache.request.quiet'] for o in six.itervalues(gets))
        assert 'memcache.response.opcode' not in gets['x']
        assert 'memcache.response.opcode' not in gets['y']

        # gets with actual return values
        gets = dict((k, v)
                    for k, v in six.iteritems(gets)
                    if k in ['k1', 'k2', 'k3'])
        assert all('memcache.response.cas_unique' in o for o in six.itervalues(gets))
        assert all(o['memcache.response.status'] == 'Success' for o in six.itervalues(gets))
        assert all((o['memcache.request.opaque'] ==
                    o['memcache.response.opaque']) for o in six.itervalues(gets))

        noop = objs[8]
        assert noop['memcache.request.command'] == 'noop'
        assert (noop['memcache.request.opaque'] ==
                noop['memcache.response.opaque'])

    def test_counter_ops(self):
        objs = self._run('memcache/memcache_bin_tcp_counter_ops.pcap')

        # all transactions succeed
        assert all(o['status'] == 'OK' for o in objs)

        set, inc, dec, get, _ = objs

        # check set command
        assert set['memcache.request.opcode'] == 'Set'
        assert set['memcache.request.command'] == 'set'
        assert set['memcache.request.type'] == 'Store'
        assert set['memcache.request.keys'] == ['cnt']
        assert set['memcache.request.exptime'] == 0
        assert set['memcache.request.bytes'] == 1
        assert set['memcache.request.count_values'] == 1
        assert set['memcache.response.opcode'] == 'Set'
        assert set['memcache.response.status'] == 'Success'
        assert 'memcache.response.cas_unique' in set
        assert (set['memcache.request.opaque'] ==
                set['memcache.response.opaque'])

        assert inc['memcache.request.opcode'] == 'Increment'
        assert inc['memcache.request.command'] == 'incr'
        assert inc['memcache.request.delta'] == 2
        assert inc['memcache.request.initial'] == 0
        assert inc['memcache.request.keys'] == ['cnt']
        assert inc['memcache.response.value'] == 2
        assert (inc['memcache.request.opaque'] ==
                inc['memcache.response.opaque'])

        assert dec['memcache.request.opcode'] == 'Decrement'
        assert dec['memcache.request.command'] == 'decr'
        assert dec['memcache.request.delta'] == 5
        assert dec['memcache.request.initial'] == 0
        assert dec['memcache.request.keys'] == ['cnt']
        assert dec['memcache.response.value'] == 0
        assert (dec['memcache.request.opaque'] ==
                dec['memcache.response.opaque'])

        assert get['memcache.request.opcode'] == 'GetK'
        assert get['memcache.request.command'] == 'get'
        assert get['memcache.request.keys'] == ['cnt']
        assert get['memcache.request.type'] == 'Load'
        assert get['memcache.response.type'] == 'Load'
        assert get['memcache.response.count_values'] == 1
        assert get['memcache.response.bytes'] == 1
        assert get['memcache.response.keys'] == ['cnt']
        assert 'memcache.response.cas_unique' in get
        assert (set['memcache.request.opaque'] ==
                set['memcache.response.opaque'])

    def test_delete(self):
        objs = self._run('memcache/memcache_bin_tcp_delete.pcap')

        set, delete, get, _ = objs

        # check set command
        assert set['status'] == 'OK'
        assert set['memcache.request.opcode'] == 'Set'
        assert set['memcache.request.command'] == 'set'
        assert set['memcache.request.type'] == 'Store'
        assert set['memcache.request.keys'] == ['key']
        assert (set['memcache.request.opaque'] ==
                set['memcache.response.opaque'])

        # check delete command
        assert delete['status'] == 'OK'
        assert delete['memcache.request.opcode'] == 'Delete'
        assert delete['memcache.request.command'] == 'delete'
        assert delete['memcache.request.type'] == 'Delete'
        assert delete['memcache.request.keys'] == ['key']
        assert delete['memcache.response.status'] == 'Success'

        # check get command on deleted key
        assert get['status'] == 'Error'
        assert get['memcache.request.command'] == 'get'
        assert get['memcache.request.type'] == 'Load'
        assert get['memcache.request.keys'] == ['key']

    def test_stats(self):
        objs = self._run('memcache/memcache_bin_tcp_stats.pcap')

        # all transactions succeed
        assert all(o['status'] == 'OK' for o in objs)

        trans = objs[2]  # get stats command

        assert trans['memcache.request.opcode'] == 'Stat'
        assert trans['memcache.request.command'] == 'stats'
        assert trans['memcache.request.type'] == 'Stats'
        assert trans['memcache.response.type'] == 'Stats'
        assert (trans['memcache.request.opaque'] ==
                trans['memcache.response.opaque'])

        # check all fields are set
        entries = trans['memcache.response.stats']
        assert all(e['name'] and e['value'] for e in entries)
