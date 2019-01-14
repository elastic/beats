from packetbeat import BaseTest

import six


class Test(BaseTest):
    """
    Basic OrientDB tests
    """

    def test_orientdb_connect(self):
        """
        Should correctly pass a OrientDB connect call
        """
        self.render_config_template(
            orientdb_ports=[2424]
        )
        self.run_packetbeat(pcap="orientdb_connect.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "orientdb"
        assert o["method"] == "connect"
        assert o["orientdb.clientName"] == "Java Impl of OrientDB Wire Protocol"
        assert o["orientdb.clientVersion"] == "v2.2.32"
        assert o["orientdb.clientID"] == "1"
        assert o["orientdb.serializationType"] == "ORecordDocument2csv"

    def test_orientdb_dbopen(self):
        """
        Should correctly pass a OrientDB dbopen call
        """
        self.render_config_template(
            orientdb_ports=[2424]
        )
        self.run_packetbeat(pcap="orientdb_dbopen.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "orientdb"
        assert o["method"] == "dbOpen"
        assert o["orientdb.clientName"] == "Java Impl of OrientDB Wire Protocol"
        assert o["orientdb.clientVersion"] == "v2.2.32"
        assert o["orientdb.clientID"] == "1"
        assert o["orientdb.database"] == "GratefulDeadConcerts"
        assert o["orientdb.serializationType"] == "ORecordDocument2csv"

    def test_orientdb_dblist(self):
        """
        Should correctly pass a OrientDB dblist call
        """
        self.render_config_template(
            orientdb_ports=[2424]
        )
        self.run_packetbeat(pcap="orientdb_dblist.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "orientdb"
        assert o["method"] == "dbList"

    def test_orientdb_dbclose(self):
        """
        Should correctly pass a OrientDB dbclose call
        """
        self.render_config_template(
            orientdb_ports=[2424]
        )
        self.run_packetbeat(pcap="orientdb_dbclose.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "orientdb"
        assert o["method"] == "dbClose"

    def test_orientdb_shutdown(self):
        """
        Should correctly pass a OrientDB shutdown call
        """
        self.render_config_template(
            orientdb_ports=[2424]
        )
        self.run_packetbeat(pcap="orientdb_shutdown.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "orientdb"
        assert o["method"] == "shutdown"

    def test_orientdb_clusteradd(self):
        """
        Should correctly pass a OrientDB clusterAdd call
        """
        self.render_config_template(
            orientdb_ports=[2424]
        )
        self.run_packetbeat(pcap="orientdb_clusteradd.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "orientdb"
        assert o["method"] == "addCluster"
        assert o["orientdb.clusterName"] == "orient_test_cluster1"
        assert o["orientdb.clusterID"] == -1

    def test_orientdb_recordcreate(self):
        """
        Should correctly pass a OrientDB record create call
        """
        self.render_config_template(
            orientdb_ports=[2424]
        )
        self.run_packetbeat(pcap="orientdb_createrecord.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "orientdb"
        assert o["method"] == "recordCreate"
        assert o["orientdb.recordType"] == 100
        assert o["orientdb.clusterID"] == 0
        assert o["orientdb.recordClass"] == "orientclass"

    def test_orientdb_recordread(self):
        """
        Should correctly pass a OrientDB record load call
        """
        self.render_config_template(
            orientdb_ports=[2424]
        )
        self.run_packetbeat(pcap="orientdb_readrecord.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "orientdb"
        assert o["method"] == "recordLoad"
        assert o["orientdb.clusterID"] == 0
        assert o["orientdb.clusterPosition"] == 3
        assert o["orientdb.fetchPlan"] == "*:0"
        assert not o["orientdb.ignoreCache"]

    def test_orientdb_recordupdate(self):
        """
        Should correctly pass a OrientDB record update call
        """
        self.render_config_template(
            orientdb_ports=[2424]
        )
        self.run_packetbeat(pcap="orientdb_updaterecord.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "orientdb"
        assert o["method"] == "recordUpdate"
        assert o["orientdb.clusterID"] == 0
        assert o["orientdb.clusterPosition"] == 3
        assert o["orientdb.updateContent"]
        assert o["orientdb.recordVersion"] == -1
        assert o["orientdb.recordType"] == "d"
        assert o["orientdb.recordClass"] == "orientclass"

    def test_orientdb_recorddelete(self):
        """
        Should correctly pass a OrientDB record delete call
        """
        self.render_config_template(
            orientdb_ports=[2424]
        )
        self.run_packetbeat(pcap="orientdb_deleterecord.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "orientdb"
        assert o["method"] == "recordDelete"
        assert o["orientdb.clusterID"] == 0
        assert o["orientdb.clusterPosition"] == 3
        assert o["orientdb.recordVersion"] == -1

    def test_orientdb_command(self):
        """
        Should correctly pass a OrientDB command call
        """
        self.render_config_template(
            orientdb_ports=[2424]
        )
        self.run_packetbeat(pcap="orientdb_command.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "orientdb"
        assert o["method"] == "commandLoad"
        assert o["orientdb.query"] == "SELECT * FROM CLUSTER:0 WHERE @rid = #0:3"
        assert o["orientdb.limit"] == "10"
        assert o["orientdb.fetchPlan"] == "*:0"
        assert o["orientdb.modByte"] == "s"
