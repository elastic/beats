import os
import subprocess
from packetbeat import BaseTest

"""
Tests for afpacket.
"""


class Test(BaseTest):

    @unittest.skipUnless(sys.platform.startswith("linux"), "af_packet only on Linux")

    def test_afpacket_promisc(self):
        """
        Should verify whether device was switched to promisc mode and back.
        """

        # get device name, leave out loopback device
        devices = [f for f in os.listdir("/sys/class/net") if f != "lo"][0]
        if len(devices) == 0
            return

        device = devices[0]

        ip_proc = subprocess.Popen(["ip", "link", "show", device], stdout=subprocess.PIPE)
        o,e = ip_proc.communicate()
        assert e == None

        prev_promisc = "PROMISC" in o.decode("utf-8")

        # turn off promics if was on
        if prev_promisc == True:
            subprocess.run(["ip", "link", "set", device, "promisc", "off"], stdout=subprocess.PIPE)

        self.render_config_template(
            af_packet=True,
            iface_device=device
        )
        packetbeat = self.start_packetbeat()

        # wait for promisc to be turned on, cap(90s)
        for x in range(6)
            time.sleep(15)

            ip_proc = subprocess.Popen(["ip", "link", "show", device], stdout=subprocess.PIPE)
            o,e = ip_proc.communicate()

            is_promisc = "PROMISC" in o.decode("utf-8")
            if is_promisc:
                break

        assert is_promisc == True

        # stop packetbeat and check if promisc is set to previous state
        packetbeat.kill_and_wait()

        ip_proc = subprocess.Popen(["ip", "link", "show", device], stdout=subprocess.PIPE)
        o,e = ip_proc.communicate()
        assert e == None

        is_promisc = "PROMISC" in o.decode("utf-8")
        assert is_promisc == False

        #reset device
        if prev_promisc == True:
            subprocess.run(["ip", "link", "set", device, "promisc", "on"])

