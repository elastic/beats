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
        assert o["event.dataset"] == "nfs"
        assert o["rpc.auth_flavor"] == "unix"
        assert "event.duration" in o
        assert "source.bytes" in o
        assert "destination.bytes" in o

        assert o["network.transport"] == "tcp"
        assert o["network.protocol"] == "nfsv3"
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
        assert o["event.dataset"] == "nfs"
        assert o["rpc.auth_flavor"] == "unix"
        assert "event.duration" in o
        assert "source.bytes" in o
        assert "destination.bytes" in o

        assert o["network.transport"] == "tcp"
        assert o["network.protocol"] == "nfsv4"
        assert o["nfs.version"] == 4
        assert o["nfs.minor_version"] == 1
        assert o["nfs.tag"] == "readdir"

        assert o["nfs.opcode"] == "READDIR"
        assert o["nfs.status"] == "NFS_OK"

    def test_first_class_op(self):
        """
        Should correctly detect first-class operation in a middle of
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

    def test_first_class_op_v42(self):
        """
        Should correctly detect first-class nfs v4.2 opration in a middle of
        compound call
        """
        self.render_config_template(
            nfs_ports=[2049],
        )
        self.run_packetbeat(pcap="nfsv42_layoutstats.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["nfs.opcode"] == "LAYOUTSTATS"

    def test_clone_notsupp_v42(self):
        """
        Should correctly detect first-class nfs v4.2 opration in a middle of
        compound call and corresponding error code
        """
        self.render_config_template(
            nfs_ports=[2049],
        )
        self.run_packetbeat(pcap="nfsv42_clone.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["nfs.opcode"] == "CLONE"
        assert o["nfs.status"] == "NFSERR_NOTSUPP"
