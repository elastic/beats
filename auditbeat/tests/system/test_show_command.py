import os
import platform
import sys
import tempfile
import unittest
from auditbeat import *


def is_root():
    if 'geteuid' not in dir(os):
        return False
    euid = os.geteuid()
    print("euid is", euid)
    return euid == 0


# Require Linux greater than 3.10
# Can't connect to kauditd in 3.10 or older
def is_supported_linux():
    p = platform.platform().split('-')
    if p[0] != 'Linux':
        return False
    kv = p[1].split('.')
    if int(kv[0]) < 3 or (int(kv[0]) == 3 and int(kv[1]) <= 10):
        return False
    return True


@unittest.skipUnless(is_supported_linux(), "Requires Linux 3.11+")
class Test(BaseTest):

    def test_show_command(self):
        """
        show sub-command is present
        Runs auditbeat show --help. The process should terminate with
        a successful status if show is recognised.
        """
        self.run_beat(extra_args=['show', '--help'], exit_code=0)

    @unittest.skipUnless(is_root(), "Requires root")
    def test_show_auditd_rules(self):
        """
        show auditd-rules sub-command
        Set some rules and read them.
        """
        pid = os.getpid()
        rules = [
            '-w {0} -p w -k rule0_{1}'.format(os.path.realpath(__file__), pid),
            '-a always,exit -S mount -F pid={0} -F key=rule1_{0}'.format(pid),
        ]
        rules_body = '|\n' + ''.join(['    ' + rule + '\n' for rule in rules])
        self.render_config_template(
            modules=[{
                "name": "auditd",
                "extras": {
                    "audit_rules": rules_body
                }
            }]
        )
        proc = self.start_beat(extra_args=['-strict.perms=false'])
        # auditbeat adds an extra rule to ignore itself
        self.wait_log_contains('Successfully added {0} of {0} audit rules.'.format(len(rules) + 1),
                               max_timeout=30)
        proc.kill()

        fd, output_file = tempfile.mkstemp()
        self.run_beat(extra_args=['show', 'auditd-rules'],
                      exit_code=0,
                      output=output_file)
        fhandle = os.fdopen(fd, 'rb')
        lines = fhandle.readlines()
        fhandle.close()
        os.unlink(output_file)
        assert len(lines) >= len(rules)
        # get rid of automatic rule
        if b'-F key=rule' not in lines[0]:
            del lines[0]

        for i in range(len(rules)):
            expected = rules[i]
            got = lines[i].strip()
            assert expected == got.decode("utf-8"), \
                "rule {0} doesn't match. expected='{1}' got='{2}'".format(
                    i, expected, got.decode("utf-8")
            )

    @unittest.skipUnless(is_root(), "Requires root")
    def test_show_auditd_status(self):
        """
        show auditd-status sub-command
        """
        expected = [
            'enabled',
            'failure',
            'pid',
            'rate_limit',
            'backlog_limit',
            'lost',
            'backlog',
            'backlog_wait_time',
            'backlog_wait_time_actual',
            'features',
        ]

        fields = dict((f, False) for f in expected)

        fd, output_file = tempfile.mkstemp()
        self.run_beat(extra_args=['show', 'auditd-status'],
                      exit_code=0,
                      output=output_file)
        fhandle = os.fdopen(fd, 'r')
        lines = fhandle.readlines()
        fhandle.close()
        os.unlink(output_file)

        for line in lines:
            if line == "PASS\n":
                break
            k, v = line.strip().split()
            assert k in fields, "Unexpected field '{0}'".format(k)
            assert not fields[k], "Field '{0}' repeated".format(k)
            n = int(v, 0)
            assert n >= 0, "Field '{0}' has negative value {1}".format(k, v)
            fields[k] = True

        for (k, v) in fields.items():
            assert v, "Field {0} not found".format(k)
