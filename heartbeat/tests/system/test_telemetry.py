from heartbeat import BaseTest
import urllib2
import json
import nose.tools


class Test(BaseTest):
    def __init__(self, *args):
        self.proc = None
        super(Test, self).__init__(*args)

    def test_telemetry(self):
        """
        Test that telemetry metrics are correctly registered and increment / decrement
        """
        server = self.start_server("hello world", 200)
        try:
            self.setup_dynamic(["-E", "http.enabled=true"])

            cfg_file = "test.yml"

            self.write_dyn_config(
                cfg_file, self.http_cfg("http://localhost:8185")
            )

            self.wait_until(lambda: self.output_has(lines=1))

            self.assert_stats(http_monitors=1, http_endpoints=1)

            tcp_hosts = ["localhost:8185", "localhost:12345"]

            self.write_dyn_config(
                cfg_file, self.tcp_cfg(*tcp_hosts)
            )

            for tcp_host in tcp_hosts:
                self.wait_until(lambda: self.log_contains(
                    "Start job 'tcp-tcp@{}".format(tcp_host)))

            init_lines = self.output_lines()
            self.wait_until(lambda: self.output_has(lines=init_lines+2))

            self.assert_stats(tcp_monitors=1, tcp_endpoints=2)
        finally:
            server.shutdown()

    def assert_stats(self, http_endpoints=0, http_monitors=0, tcp_endpoints=0, tcp_monitors=0, icmp_endpoints=0, icmp_monitors=0):
        total_monitors = http_monitors+tcp_monitors+icmp_monitors

        stats = json.loads(urllib2.urlopen(
            "http://localhost:5066/stats").read())
        nose.tools.assert_dict_equal(stats['heartbeat'], {
            'monitors': total_monitors,
            'http': {
                'monitors': http_monitors,
                'endpoints': http_endpoints,
            },
            'tcp': {
                'monitors': tcp_monitors,
                'endpoints': tcp_endpoints,
            },
            'icmp': {
                'monitors': icmp_monitors,
                'endpoints': icmp_endpoints,
            }
        })
