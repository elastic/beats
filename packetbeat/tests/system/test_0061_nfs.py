from packetbeat import BaseTest

"""
Tests for the NFS
"""


class Test(BaseTest):
    def test_V3(self):
        """
        Should correctly parse NFS v3 packet
        """
        self.render_config_template(
            nfs_ports=[2049],
        )
        self.run_packetbeat(pcap="nfs_v3.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["type"] == "nfs"
        assert o["rpc.auth_flavor"] == "unix"
        assert "rpc.time" in o
        assert "rpc.time_str" in o
        assert "rpc.call_size" in o
        assert "rpc.reply_size" in o

        assert o["nfs.version"] == 3
        assert o["nfs.opcode"] == "LOOKUP"
        assert o["nfs.status"] == "NFSERR_NOENT"

    def test_v4(self):
        """
        Should correctly parse NFSv4.1 packet
        """
        self.render_config_template(
            nfs_ports=[2049],
        )
        self.run_packetbeat(pcap="nfs_v4.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["type"] == "nfs"
        assert o["rpc.auth_flavor"] == "unix"
        assert "rpc.time" in o
        assert "rpc.time_str" in o
        assert "rpc.call_size" in o
        assert "rpc.reply_size" in o

        assert o["nfs.version"] == 4
        assert o["nfs.minor_version"] == 1
        assert o["nfs.tag"] == "readdir"

        assert o["nfs.opcode"] == "READDIR"
        assert o["nfs.status"] == "NFS_OK"

    def test_first_class_op(self):
        """
        Should cerrectly detect first-class opration in a middle of
        compound call
        """
        self.render_config_template(
            nfs_ports=[2049],
        )
        self.run_packetbeat(pcap="nfs4_close.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["nfs.opcode"] == "CLOSE"
