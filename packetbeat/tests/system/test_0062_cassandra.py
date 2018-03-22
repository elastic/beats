from packetbeat import BaseTest

"""
Tests for the Cassandra
"""


class Test(BaseTest):

    def test_create_keyspace(self):
        """
        Should correctly create a keyspace in Cassandra
        """
        self.render_config_template(
            cassandra_ports=[9042],
            cassandra_send_request=True,
            cassandra_send_response=True,
            cassandra_send_request_header=True,
            cassandra_send_response_header=True,
        )
        self.run_packetbeat(pcap="cassandra/v4/cassandra_create_keyspace.pcap", debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]

        assert o["type"] == "cassandra"
        assert o["port"] == 9042

        assert o[
            "cassandra.request.query"] == "CREATE KEYSPACE mykeyspace WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 };"
        assert o["cassandra.request.headers.version"] == "4"
        assert o["cassandra.request.headers.op"] == "QUERY"
        assert o["cassandra.request.headers.length"] == 124
        assert o["cassandra.request.headers.flags"] == "Default"
        assert o["cassandra.request.headers.stream"] == 20

        assert o["cassandra.response.result.type"] == "schemaChanged"
        assert o["cassandra.response.result.schema_change.change"] == "CREATED"
        assert o["cassandra.response.result.schema_change.keyspace"] == "mykeyspace"
        assert o["cassandra.response.result.schema_change.target"] == "KEYSPACE"

        assert o["cassandra.response.headers.version"] == "4"
        assert o["cassandra.response.headers.length"] == 35
        assert o["cassandra.response.headers.op"] == "RESULT"
        assert o["cassandra.response.headers.flags"] == "Default"
        assert o["cassandra.response.headers.stream"] == 20

    def test_create_table(self):
        """
       Should correctly create a table in Cassandra
       """
        self.render_config_template(
            cassandra_ports=[9042],
            cassandra_send_request=True,
            cassandra_send_response=True,
            cassandra_send_request_header=True,
            cassandra_send_response_header=True,
        )
        self.run_packetbeat(pcap="cassandra/v4/cassandra_create_table.pcap", debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "cassandra"
        assert o["port"] == 9042

        assert o[
            "cassandra.request.query"] == "CREATE TABLE users (\n  user_id int PRIMARY KEY,\n  fname text,\n  lname text\n);"

        assert o["cassandra.request.headers.version"] == "4"
        assert o["cassandra.request.headers.op"] == "QUERY"
        assert o["cassandra.request.headers.length"] == 98
        assert o["cassandra.request.headers.flags"] == "Default"
        assert o["cassandra.request.headers.stream"] == 49

        assert o["cassandra.response.result.type"] == "schemaChanged"
        assert o["cassandra.response.headers.version"] == "4"
        assert o["cassandra.response.headers.length"] == 39
        assert o["cassandra.response.headers.op"] == "RESULT"
        assert o["cassandra.response.headers.flags"] == "Default"
        assert o["cassandra.response.headers.stream"] == 49

    def test_insert_data(self):
        """
       Should correctly insert record into table in Cassandra
       """
        self.render_config_template(
            cassandra_ports=[9042],
            cassandra_send_request=True,
            cassandra_send_response=True,
            cassandra_send_request_header=True,
            cassandra_send_response_header=True,
        )
        self.run_packetbeat(pcap="cassandra/v4/cassandra_insert.pcap", debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        print(o)
        assert o["type"] == "cassandra"
        assert o["port"] == 9042

        assert o[
            "cassandra.request.query"] == "INSERT INTO users (user_id,  fname, lname)\n  VALUES (1745, 'john', 'smith');"
        assert o["cassandra.request.headers.version"] == "4"
        assert o["cassandra.request.headers.op"] == "QUERY"
        assert o["cassandra.request.headers.length"] == 97
        assert o["cassandra.request.headers.flags"] == "Default"
        assert o["cassandra.request.headers.stream"] == 252

        assert o["cassandra.response.result.type"] == "void"
        assert o["cassandra.response.headers.version"] == "4"
        assert o["cassandra.response.headers.length"] == 4
        assert o["cassandra.response.headers.op"] == "RESULT"
        assert o["cassandra.response.headers.flags"] == "Default"
        assert o["cassandra.response.headers.stream"] == 252

    def test_select_data(self):
        """
       Should correctly select record from table in Cassandra
       """
        self.render_config_template(
            cassandra_ports=[9042],
            cassandra_send_request=True,
            cassandra_send_response=True,
            cassandra_send_request_header=True,
            cassandra_send_response_header=True,
        )
        self.run_packetbeat(pcap="cassandra/v4/cassandra_select.pcap", debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "cassandra"
        assert o["port"] == 9042

        assert o["cassandra.request.query"] == "SELECT * FROM users;"
        assert o["cassandra.request.headers.version"] == "4"
        assert o["cassandra.request.headers.op"] == "QUERY"
        assert o["cassandra.request.headers.length"] == 41
        assert o["cassandra.request.headers.flags"] == "Default"
        assert o["cassandra.request.headers.stream"] == 253

        assert o["cassandra.response.result.type"] == "rows"
        assert o["cassandra.response.headers.version"] == "4"
        assert o["cassandra.response.headers.length"] == 89
        assert o["cassandra.response.headers.op"] == "RESULT"
        assert o["cassandra.response.headers.flags"] == "Default"
        assert o["cassandra.response.headers.stream"] == 253

    def test_create_index(self):
        """
       Should correctly create index of table in Cassandra
       """
        self.render_config_template(
            cassandra_ports=[9042],
            cassandra_send_request=True,
            cassandra_send_response=True,
            cassandra_send_request_header=True,
            cassandra_send_response_header=True,
        )
        self.run_packetbeat(pcap="cassandra/v4/cassandra_create_index.pcap", debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "cassandra"
        assert o["port"] == 9042

        assert o["cassandra.request.query"] == "CREATE INDEX ON users (lname);"
        assert o["cassandra.request.headers.version"] == "4"
        assert o["cassandra.request.headers.op"] == "QUERY"
        assert o["cassandra.request.headers.length"] == 51
        assert o["cassandra.request.headers.flags"] == "Default"
        assert o["cassandra.request.headers.stream"] == 92

        assert o["cassandra.response.result.type"] == "schemaChanged"

        assert o["cassandra.response.headers.version"] == "4"
        assert o["cassandra.response.headers.length"] == 39
        assert o["cassandra.response.headers.op"] == "RESULT"
        assert o["cassandra.response.headers.flags"] == "Default"
        assert o["cassandra.response.headers.stream"] == 92

    def test_trace_error(self):
        """
       Should correctly catch a error message and trace flag was enabled
       """
        self.render_config_template(
            cassandra_ports=[9042],
            cassandra_send_request=True,
            cassandra_send_response=True,
            cassandra_send_request_header=True,
            cassandra_send_response_header=True,
        )
        self.run_packetbeat(pcap="cassandra/v4/cassandra_trace_err.pcap", debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "cassandra"
        assert o["port"] == 9042

        assert o["bytes_in"] == 55
        assert o["bytes_out"] == 62
        assert o["cassandra.request.query"] == "DROP KEYSPACE mykeyspace;"
        print(o)

        assert o["cassandra.request.headers.version"] == "4"
        assert o["cassandra.request.headers.op"] == "QUERY"
        assert o["cassandra.request.headers.length"] == 46
        assert o["cassandra.request.headers.flags"] == "Tracing"
        assert o["cassandra.request.headers.stream"] == 275

        assert o["cassandra.response.error.code"] == 8960
        assert o["cassandra.response.error.msg"] == "Cannot drop non existing keyspace 'mykeyspace'."
        assert o["cassandra.response.error.type"] == "errConfig"

        assert o["cassandra.response.headers.version"] == "4"
        assert o["cassandra.response.headers.length"] == 53
        assert o["cassandra.response.headers.op"] == "ERROR"
        assert o["cassandra.response.headers.flags"] == "Default"
        assert o["cassandra.response.headers.stream"] == 275

    def test_select_use_index(self):
        """
       Should correctly select record from table (use index) in Cassandra
       """
        self.render_config_template(
            cassandra_ports=[9042],
            cassandra_send_request=True,
            cassandra_send_response=True,
            cassandra_send_request_header=True,
            cassandra_send_response_header=True,
        )
        self.run_packetbeat(pcap="cassandra/v4/cassandra_select_via_index.pcap", debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        print(o)
        assert o["type"] == "cassandra"
        assert o["port"] == 9042

        assert o["cassandra.request.query"] == "SELECT * FROM users WHERE lname = 'smith';"

        assert o["cassandra.request.headers.version"] == "4"
        assert o["cassandra.request.headers.op"] == "QUERY"
        assert o["cassandra.request.headers.length"] == 63
        assert o["cassandra.request.headers.flags"] == "Default"
        assert o["cassandra.request.headers.stream"] == 262

        assert o["cassandra.response.headers.version"] == "4"
        assert o["cassandra.response.headers.length"] == 89
        assert o["cassandra.response.headers.op"] == "RESULT"
        assert o["cassandra.response.headers.flags"] == "Default"
        assert o["cassandra.response.headers.stream"] == 262
        assert o["cassandra.response.result.type"] == "rows"

    def test_ops_mixed(self):
        """
       Should correctly have mixed operation happened in Cassandra
       """
        self.render_config_template(
            cassandra_ports=[9042],
            cassandra_send_request=True,
            cassandra_send_response=True,
            cassandra_send_request_header=True,
            cassandra_send_response_header=True,
        )

        self.run_packetbeat(pcap="cassandra/v4/cassandra_mixed_frame.pcap", debug_selectors=["*"])
        objs = self.read_output()

        o = objs[0]
        print(o)
        assert o["type"] == "cassandra"
        assert o["port"] == 9042
        assert o["bytes_in"] == 9
        assert o["bytes_out"] == 61

        assert o["cassandra.request.headers.version"] == "4"
        assert o["cassandra.request.headers.op"] == "OPTIONS"
        assert o["cassandra.request.headers.length"] == 0
        assert o["cassandra.request.headers.flags"] == "Default"
        assert o["cassandra.request.headers.stream"] == 0

        assert o["cassandra.response.headers.version"] == "4"
        assert o["cassandra.response.headers.length"] == 52
        assert o["cassandra.response.headers.op"] == "SUPPORTED"
        assert o["cassandra.response.headers.flags"] == "Default"
        assert o["cassandra.response.headers.stream"] == 0

        o = objs[1]
        print(o)
        assert o["type"] == "cassandra"
        assert o["port"] == 9042
        assert o["bytes_in"] == 31
        assert o["bytes_out"] == 9

        assert o["cassandra.request.headers.version"] == "4"
        assert o["cassandra.request.headers.op"] == "STARTUP"
        assert o["cassandra.request.headers.length"] == 22
        assert o["cassandra.request.headers.flags"] == "Default"
        assert o["cassandra.request.headers.stream"] == 1

        assert o["cassandra.response.headers.version"] == "4"
        assert o["cassandra.response.headers.length"] == 0
        assert o["cassandra.response.headers.op"] == "READY"
        assert o["cassandra.response.headers.flags"] == "Default"
        assert o["cassandra.response.headers.stream"] == 1

        o = objs[2]
        print(o)
        assert o["type"] == "cassandra"
        assert o["port"] == 9042
        assert o["bytes_in"] == 58
        assert o["bytes_out"] == 9

        assert o["cassandra.request.headers.version"] == "4"
        assert o["cassandra.request.headers.op"] == "REGISTER"
        assert o["cassandra.request.headers.length"] == 49
        assert o["cassandra.request.headers.flags"] == "Default"
        assert o["cassandra.request.headers.stream"] == 2

        assert o["cassandra.response.headers.version"] == "4"
        assert o["cassandra.response.headers.length"] == 0
        assert o["cassandra.response.headers.op"] == "READY"
        assert o["cassandra.response.headers.flags"] == "Default"
        assert o["cassandra.response.headers.stream"] == 2

    def test_ops_ignored(self):
        """
       Should correctly ignore OPTIONS and REGISTER operation
       """
        self.render_config_template(
            cassandra_ports=[9042],
            cassandra_send_request=True,
            cassandra_send_response=True,
            cassandra_send_request_header=True,
            cassandra_send_response_header=True,
            cassandra_ignored_ops=["OPTIONS", "REGISTER"]
        )

        self.run_packetbeat(pcap="cassandra/v4/cassandra_mixed_frame.pcap", debug_selectors=["*"])
        objs = self.read_output()

        o = objs[0]
        assert o["type"] == "cassandra"
        assert o["port"] == 9042
        assert o["bytes_in"] == 31
        assert o["bytes_out"] == 9

        assert o["cassandra.request.headers.version"] == "4"
        assert o["cassandra.request.headers.op"] == "STARTUP"
        assert o["cassandra.request.headers.length"] == 22
        assert o["cassandra.request.headers.flags"] == "Default"
        assert o["cassandra.request.headers.stream"] == 1

        assert o["cassandra.response.headers.version"] == "4"
        assert o["cassandra.response.headers.length"] == 0
        assert o["cassandra.response.headers.op"] == "READY"
        assert o["cassandra.response.headers.flags"] == "Default"
        assert o["cassandra.response.headers.stream"] == 1

        o = objs[1]
        assert o["type"] == "cassandra"
        assert o["port"] == 9042
        assert o["bytes_in"] == 101
        assert o["bytes_out"] == 116

        assert o["cassandra.request.headers.version"] == "4"
        assert o["cassandra.request.headers.op"] == "QUERY"
        assert o["cassandra.request.headers.length"] == 92
        assert o["cassandra.request.headers.flags"] == "Default"
        assert o["cassandra.request.headers.stream"] == 3

        assert o["cassandra.response.headers.version"] == "4"
        assert o["cassandra.response.headers.length"] == 107
        assert o["cassandra.response.headers.op"] == "RESULT"
        assert o["cassandra.response.headers.flags"] == "Default"
        assert o["cassandra.response.headers.stream"] == 3

    def test_compressed_frame(self):
        """
       Should correctly have some compressed frame should happened in Cassandra
       """
        self.render_config_template(
            cassandra_ports=[9042],
            cassandra_send_request=True,
            cassandra_send_response=True,
            cassandra_send_request_header=True,
            cassandra_send_response_header=True,
            cassandra_compressor="snappy",
        )

        self.run_packetbeat(pcap="cassandra/v4/cassandra_compressed.pcap", debug_selectors=["*"])
        objs = self.read_output()

        o = objs[0]
        print(o)
        assert o["type"] == "cassandra"
        assert o["port"] == 9042
        assert o["bytes_in"] == 52
        assert o["bytes_out"] == 10

        assert o["cassandra.request.headers.version"] == "4"
        assert o["cassandra.request.headers.op"] == "STARTUP"
        assert o["cassandra.request.headers.length"] == 43
        assert o["cassandra.request.headers.flags"] == "Default"
        assert o["cassandra.request.headers.stream"] == 0

        assert o["cassandra.response.headers.version"] == "4"
        assert o["cassandra.response.headers.length"] == 1
        assert o["cassandra.response.headers.op"] == "READY"
        assert o["cassandra.response.headers.flags"] == "Compress"
        assert o["cassandra.response.headers.stream"] == 0

        o = objs[1]
        print(o)
        assert o["type"] == "cassandra"
        assert o["port"] == 9042
        assert o["bytes_in"] == 53
        assert o["bytes_out"] == 10

        assert o["cassandra.request.headers.version"] == "4"
        assert o["cassandra.request.headers.op"] == "REGISTER"
        assert o["cassandra.request.headers.length"] == 44
        assert o["cassandra.request.headers.flags"] == "Compress"
        assert o["cassandra.request.headers.stream"] == 64

        assert o["cassandra.response.headers.version"] == "4"
        assert o["cassandra.response.headers.length"] == 1
        assert o["cassandra.response.headers.op"] == "READY"
        assert o["cassandra.response.headers.flags"] == "Compress"
        assert o["cassandra.response.headers.stream"] == 64

        o = objs[2]
        print(o)
        assert o["type"] == "cassandra"
        assert o["port"] == 9042
        assert o["bytes_in"] == 62
        assert o["bytes_out"] == 165

        assert o["cassandra.request.query"] == "SELECT * FROM system.local WHERE key='local'"

        assert o["cassandra.request.headers.version"] == "4"
        assert o["cassandra.request.headers.op"] == "QUERY"
        assert o["cassandra.request.headers.length"] == 53
        assert o["cassandra.request.headers.flags"] == "Compress"
        assert o["cassandra.request.headers.stream"] == 0

        assert o["cassandra.response.headers.version"] == "4"
        assert o["cassandra.response.headers.length"] == 156
        assert o["cassandra.response.headers.op"] == "RESULT"
        assert o["cassandra.response.headers.flags"] == "Compress"
        assert o["cassandra.response.headers.stream"] == 64
        assert o["cassandra.response.result.type"] == "rows"
        assert o["cassandra.response.result.rows.num_rows"] == 290917
        assert o["cassandra.response.result.rows.meta.col_count"] == 9
        assert o["cassandra.response.result.rows.meta.flags"] == "GlobalTableSpec"
        assert o["cassandra.response.result.rows.meta.keyspace"] == "system"
        assert o["cassandra.response.result.rows.meta.table"] == "peers"
