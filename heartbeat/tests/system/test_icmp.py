import os
import unittest
import platform
import socket
import sys
from heartbeat import BaseTest
from elasticsearch import Elasticsearch
from beat.beat import INTEGRATION_TESTS
import nose.tools
import logging
import subprocess


class Test(BaseTest):

    def has_group_permission(self):
        runningUser = subprocess.check_output(['whoami']).strip()
        runningGroups = subprocess.check_output(
            ['id', '-G', runningUser]).strip()
        runningGroups = runningGroups.split(" ")
        runningGroups = map(int, runningGroups)
        runningGroups.sort()
        print(runningGroups)
        result = subprocess.check_output(
            ['sysctl', 'net.ipv4.ping_group_range']).strip()
        result = result.split("= ")
        result = result[1].split("\t")
        result = map(int, result)
        firstGroup = result[0]
        lastGroup = result[1]
        if any(firstGroup == group for group in runningGroups):
            return (True)
        if any(lastGroup > group for group in runningGroups):
            return (True)

        return (False)

    def has_admin(self):
        if os.name == 'nt':
            try:
                # only windows users with admin privileges can read the C:\windows\temp
                temp = os.listdir(os.sep.join(
                    [os.environ.get('SystemRoot', 'C:\\windows'), 'temp']))
            except:
                return (False)
            else:
                return (True)
        else:
            if 'SUDO_USER' in os.environ and os.geteuid() == 0:
                return (True)
            else:
                return (False)

    def test_base(self):
        """
        Basic test with icmp root non privilege ICMP test.

        """

        config = {
            "monitors": [
                {
                    "type": "icmp",
                    "schedule": "*/5 * * * * * *",
                    "hosts": ["8.8.8.8"],
                }
            ]
        }

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            **config
        )

        adminRights = self.has_admin()
        if adminRights == True:
            proc = self.start_beat()
            self.wait_until(lambda: self.log_contains(
                "heartbeat is running"))
            proc.check_kill_and_wait()
        else:
            if platform.system() in ["Linux", "Darwin"]:
                groupRights = self.has_group_permission()
                if groupRights == True:
                    proc = self.start_beat()
                    self.wait_until(lambda: self.log_contains(
                        "heartbeat is running"))
                    proc.check_kill_and_wait()
                else:
                    exit_code = self.run_beat()
                    assert exit_code == 1
                    assert self.wait_until(
                        lambda: self.log_contains(
                            "You dont have root permission to run ping"),
                        max_timeout=10)
            else:
        # windows seems to allow all users to run sockets
                proc = self.start_beat()
                self.wait_until(lambda: self.log_contains(
                    "heartbeat is running"))
                proc.check_kill_and_wait()
