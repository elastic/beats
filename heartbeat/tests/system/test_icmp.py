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
import time


class Test(BaseTest):

    def has_group_permission(self):

        try:
            runningUser = subprocess.check_output(['whoami']).strip()
            runningGroups = subprocess.check_output(
                ['id', '-G', runningUser]).strip()
            runningGroups = runningGroups.split(" ")
            runningGroups = map(int, runningGroups)
            runningGroups.sort()
            sys.stderr.write("RUNNING GROUPS: %s\n" % runningGroups)
            result = subprocess.check_output(
                ['sysctl', 'net.ipv4.ping_group_range']).strip()
            sys.stderr.write("GROUP RANGE: %s\n" % result)
            result = result.split("= ")
            result = result[1].split("\t")
            result = map(int, result)
            firstGroup = result[0]
            lastGroup = result[1]
            if any(firstGroup == group for group in runningGroups):
                return (True)
            if any(lastGroup > group for group in runningGroups):
                return (True)
        except subprocess.CalledProcessError, e:
            sys.stderr.write("Error trying sysctl: %s\n" % e.output)

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
            sys.stderr.write("EUID: %s\n" % os.geteuid())
            if os.geteuid() == 0:
                return (True)
            else:
                return (False)

    def test_base(self):
        """
        Basic test with icmp root non privilege ICMP test.

        """
        sys.stderr.write("STARTING ICMP TEST\n")

        config = {
            "monitors": [
                {
                    "type": "icmp",
                    "schedule": "*/5 * * * * * *",
                    "hosts": ["8.8.8.8"],
                }
            ]
        }

        sys.stderr.write("WROTE THE THING")
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            **config
        )

        sys.stderr.write("DETECTED PLATFORM\n")
        if platform.system() in ["Linux", "Darwin"]:
            sys.stderr.write("LINUX\n")
            adminRights = self.has_admin()
            groupRights = self.has_group_permission()
            if groupRights == True or adminRights == True:
                sys.stderr.write("STARTING BEAT\n\n")
                proc = self.start_beat()
                sys.stderr.write("STARTED BEAT\n")
                sys.stderr.write("HAS GROUP OR ADMIN RIGHTS WAITING FOR HB\n")
                self.wait_until(lambda: self.log_contains("heartbeat is running"))
                sys.stderr.write("CHECK WAIT %s\n")
                proc.check_kill_and_wait()
            else:
                sys.stderr.write("NO RIGHTS\n")
                proc = self.start_beat()
                expected = "You dont have root permission to run ping"
                self.wait_until(lambda: self.log_contains(expected), 30)
                sys.stderr.write("RAN IT\n")
        else:
            sys.stderr.write("ON WINDOWS %s\n")
            # windows seems to allow all users to run sockets
            proc = self.start_beat()
            self.wait_until(lambda: self.log_contains(
                "heartbeat is running"))
            proc.check_kill_and_wait()
        sys.stderr.write("FINISHED ICMP TEST\n")
