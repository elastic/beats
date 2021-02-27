import os
import subprocess
import sys
import time
import unittest
from packetbeat import BaseTest

"""
Tests for afpacket.
"""


def is_root():
    if 'geteuid' not in dir(os):
        return False
    euid = os.geteuid()
    print("euid is", euid)
    return euid == 0


class Test(BaseTest):

    @unittest.skipUnless(
        sys.platform.startswith("linux"),
        "af_packet only on Linux")
    @unittest.skipUnless(is_root(), "Requires root")
    def test_afpacket_promisc(self):
        """
        Should switch to promisc mode and back.
        """

        # get device name, leave out loopback device
        devices = [f for f in os.listdir(
            "/sys/class/net") if f.startswith("lo")]
        assert len(devices) > 0

        device = devices[0]

        ip_proc = subprocess.Popen(
            ["ip", "link", "show", device], stdout=subprocess.PIPE)
        o, e = ip_proc.communicate()
        assert e is None

        prev_promisc = "PROMISC" in o.decode("utf-8")

        # turn off promics if was on
        if prev_promisc:
            subprocess.call(["ip", "link", "set", device,
                             "promisc", "off"], stdout=subprocess.PIPE)

        self.render_config_template(
            af_packet=True,
            iface_device=device
        )
        packetbeat = self.start_packetbeat()

        # wait for promisc to be turned on, cap(90s)
        for x in range(10):
            time.sleep(5)

            ip_proc = subprocess.Popen(
                ["ip", "link", "show", device], stdout=subprocess.PIPE)
            o, e = ip_proc.communicate()

            is_promisc = "PROMISC" in o.decode("utf-8")
            if is_promisc:
                break

        assert is_promisc

        # stop packetbeat and check if promisc is set to previous state
        packetbeat.kill_and_wait()

        ip_proc = subprocess.Popen(
            ["ip", "link", "show", device], stdout=subprocess.PIPE)
        o, e = ip_proc.communicate()
        assert e is None

        is_promisc = "PROMISC" in o.decode("utf-8")
        assert is_promisc == False

        # reset device
        if prev_promisc:
            subprocess.call(["ip", "link", "set", device, "promisc", "on"])
