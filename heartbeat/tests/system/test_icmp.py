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


class Test(BaseTest):

    def setUp(self):
        logging.basicConfig(format='%(asctime)s %(levelname)s %(message)s', datefmt='%Y-%m-%d %I:%M:%S',
                            level=logging.INFO)

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

        logging.info("Begin testing")

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            **config
        )
        logging.info("Begin has_admin")
        adminRights = self.has_admin()
        logging.info("End has_admin")
        if adminRights == True:
            logging.info("adminRights == True")
            proc = self.start_beat()
            self.wait_until(lambda: self.log_contains(
                "heartbeat is running"))
            proc.check_kill_and_wait()
        else:
            logging.info("adminRights == False")
            if platform.system() in ["Linux", "Darwin"]:
                logging.info("platform.system() in Linux Darwin")
                exit_code = self.run_beat()
                assert exit_code == 1
                assert self.log_contains(
                    "You dont have root permission to run ping")
            else:
                logging.info("platform.system() in Else")
                # windows seems to allow all users to run sockets
                proc = self.start_beat()
                self.wait_until(lambda: self.log_contains(
                    "heartbeat is running"))
                proc.check_kill_and_wait()
