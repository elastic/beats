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
        self.run_packetbeat(pcap="cassandra/v4/cassandra_create_keyspace.pcap",debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]

        assert o["type"] == "cassandra"
        assert o["port"] == 9042

        q=o["cassandra_request"]
        assert q["query"] == "CREATE KEYSPACE mykeyspace WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 };"

        h= q["request_headers"]
        assert h["version"] == "4"
        assert h["op"] == "QUERY"
        assert h["length"] == 124
        assert h["flags"] == "FLAG_0"
        assert h["stream"] == 20


        r=o["cassandra_response"]
        assert r["result_type"]=="schemaChanged"

        h2= r["response_headers"]
        assert h2["version"] == "4"
        assert h2["length"] == 35
        assert h2["op"] == "RESULT"
        assert h2["flags"] == "FLAG_0"
        assert h2["stream"] == 20

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
        self.run_packetbeat(pcap="cassandra/v4/cassandra_create_table.pcap",debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "cassandra"
        assert o["port"] == 9042

        q=o["cassandra_request"]
        assert q["query"] == "CREATE TABLE users (\n  user_id int PRIMARY KEY,\n  fname text,\n  lname text\n);"

        h= q["request_headers"]
        assert h["version"] == "4"
        assert h["op"] == "QUERY"
        assert h["length"] == 98
        assert h["flags"] == "FLAG_0"
        assert h["stream"] == 49


        r=o["cassandra_response"]
        assert r["result_type"]=="schemaChanged"
        h2= r["response_headers"]
        assert h2["version"] == "4"
        assert h2["length"] == 39
        assert h2["op"] == "RESULT"
        assert h2["flags"] == "FLAG_0"
        assert h2["stream"] == 49

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
        self.run_packetbeat(pcap="cassandra/v4/cassandra_insert.pcap",debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        print o
        assert o["type"] == "cassandra"
        assert o["port"] == 9042

        q=o["cassandra_request"]
        assert q["query"] == "INSERT INTO users (user_id,  fname, lname)\n  VALUES (1745, 'john', 'smith');"
        h= q["request_headers"]
        assert h["version"] == "4"
        assert h["op"] == "QUERY"
        assert h["length"] == 97
        assert h["flags"] == "FLAG_0"
        assert h["stream"] == 252

        r=o["cassandra_response"]
        assert r["result_type"]=="void"
        h2= r["response_headers"]
        assert h2["version"] == "4"
        assert h2["length"] == 4
        assert h2["op"] == "RESULT"
        assert h2["flags"] == "FLAG_0"
        assert h2["stream"] == 252

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
        self.run_packetbeat(pcap="cassandra/v4/cassandra_select.pcap",debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "cassandra"
        assert o["port"] == 9042

        q=o["cassandra_request"]
        assert q["query"] == "SELECT * FROM users;"
        h= q["request_headers"]
        assert h["version"] == "4"
        assert h["op"] == "QUERY"
        assert h["length"] == 41
        assert h["flags"] == "FLAG_0"
        assert h["stream"] == 253


        r=o["cassandra_response"]
        assert r["result_type"]=="rows"
        h2= r["response_headers"]
        assert h2["version"] == "4"
        assert h2["length"] == 89
        assert h2["op"] == "RESULT"
        assert h2["flags"] == "FLAG_0"
        assert h2["stream"] == 253

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
        self.run_packetbeat(pcap="cassandra/v4/cassandra_create_index.pcap",debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "cassandra"
        assert o["port"] == 9042

        q=o["cassandra_request"]
        assert q["query"] == "CREATE INDEX ON users (lname);"
        h= q["request_headers"]
        assert h["version"] == "4"
        assert h["op"] == "QUERY"
        assert h["length"] == 51
        assert h["flags"] == "FLAG_0"
        assert h["stream"] == 92

        r=o["cassandra_response"]
        assert r["result_type"]=="schemaChanged"
        h2= r["response_headers"]
        assert h2["version"] == "4"
        assert h2["length"] == 39
        assert h2["op"] == "RESULT"
        assert h2["flags"] == "FLAG_0"
        assert h2["stream"] == 92

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
        self.run_packetbeat(pcap="cassandra/v4/cassandra_trace_err.pcap",debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        assert o["type"] == "cassandra"
        assert o["port"] == 9042

        q=o["cassandra_request"]
        assert o["bytes_in"] == 55
        assert o["bytes_out"] == 62
        assert q["query"] == "DROP KEYSPACE mykeyspace;"
        h= q["request_headers"]
        assert h["version"] == "4"
        assert h["op"] == "QUERY"
        assert h["length"] == 46
        assert h["flags"] == "Tracing"
        assert h["stream"] == 275

        r=o["cassandra_response"]
        assert r["err_code"]==8960
        assert r["err_msg"]=="Cannot drop non existing keyspace 'mykeyspace'."
        assert r["err_type"]=="errConfig"
        h2= r["response_headers"]
        assert h2["version"] == "4"
        assert h2["length"] == 53
        assert h2["op"] == "ERROR"
        assert h2["flags"] == "FLAG_0"
        assert h2["stream"] == 275

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
        self.run_packetbeat(pcap="cassandra/v4/cassandra_select_via_index.pcap",debug_selectors=["*"])
        objs = self.read_output()
        o = objs[0]
        print o
        assert o["type"] == "cassandra"
        assert o["port"] == 9042

        q=o["cassandra_request"]
        assert q["query"] == "SELECT * FROM users WHERE lname = 'smith';"
        h= q["request_headers"]
        assert h["version"] == "4"
        assert h["op"] == "QUERY"
        assert h["length"] == 63
        assert h["flags"] == "FLAG_0"
        assert h["stream"] == 262


        r=o["cassandra_response"]
        h2= r["response_headers"]
        assert h2["version"] == "4"
        assert h2["length"] == 89
        assert h2["op"] == "RESULT"
        assert h2["flags"] == "FLAG_0"
        assert h2["stream"] == 262
        assert r["result_type"]=="rows"

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

        self.run_packetbeat(pcap="cassandra/v4/cassandra_mixed_frame.pcap",debug_selectors=["*"])
        objs = self.read_output()

        o = objs[0]
        print o
        assert o["type"] == "cassandra"
        assert o["port"] == 9042
        assert o["bytes_in"] == 9
        assert o["bytes_out"] == 61

        q=o["cassandra_request"]
        h= q["request_headers"]
        assert h["version"] == "4"
        assert h["op"] == "OPTIONS"
        assert h["length"] == 0
        assert h["flags"] == "FLAG_0"
        assert h["stream"] == 0


        r=o["cassandra_response"]
        h2= r["response_headers"]
        assert h2["version"] == "4"
        assert h2["length"] == 52
        assert h2["op"] == "SUPPORTED"
        assert h2["flags"] == "FLAG_0"
        assert h2["stream"] == 0

        o = objs[1]
        print o
        assert o["type"] == "cassandra"
        assert o["port"] == 9042
        assert o["bytes_in"] == 31
        assert o["bytes_out"] == 9

        q=o["cassandra_request"]
        h= q["request_headers"]
        assert h["version"] == "4"
        assert h["op"] == "STARTUP"
        assert h["length"] == 22
        assert h["flags"] == "FLAG_0"
        assert h["stream"] == 1


        r=o["cassandra_response"]
        h2= r["response_headers"]
        assert h2["version"] == "4"
        assert h2["length"] == 0
        assert h2["op"] == "READY"
        assert h2["flags"] == "FLAG_0"
        assert h2["stream"] == 1

        o = objs[2]
        print o
        assert o["type"] == "cassandra"
        assert o["port"] == 9042
        assert o["bytes_in"] == 58
        assert o["bytes_out"] == 9

        q=o["cassandra_request"]
        h= q["request_headers"]
        assert h["version"] == "4"
        assert h["op"] == "REGISTER"
        assert h["length"] == 49
        assert h["flags"] == "FLAG_0"
        assert h["stream"] == 2


        r=o["cassandra_response"]
        h2= r["response_headers"]
        assert h2["version"] == "4"
        assert h2["length"] == 0
        assert h2["op"] == "READY"
        assert h2["flags"] == "FLAG_0"
        assert h2["stream"] == 2

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
            cassandra_ignored_ops= ["OPTIONS","REGISTER"]
        )

        self.run_packetbeat(pcap="cassandra/v4/cassandra_mixed_frame.pcap",debug_selectors=["*"])
        objs = self.read_output()

        o = objs[0]
        print o
        assert o["type"] == "cassandra"
        assert o["port"] == 9042
        assert o["bytes_in"] == 31
        assert o["bytes_out"] == 9

        q=o["cassandra_request"]
        h= q["request_headers"]
        assert h["version"] == "4"
        assert h["op"] == "STARTUP"
        assert h["length"] == 22
        assert h["flags"] == "FLAG_0"
        assert h["stream"] == 1


        r=o["cassandra_response"]
        h2= r["response_headers"]
        assert h2["version"] == "4"
        assert h2["length"] == 0
        assert h2["op"] == "READY"
        assert h2["flags"] == "FLAG_0"
        assert h2["stream"] == 1

        o = objs[1]
        print o
        assert o["type"] == "cassandra"
        assert o["port"] == 9042
        assert o["bytes_in"] == 101
        assert o["bytes_out"] == 116

        q=o["cassandra_request"]
        h= q["request_headers"]
        assert h["version"] == "4"
        assert h["op"] == "QUERY"
        assert h["length"] == 92
        assert h["flags"] == "FLAG_0"
        assert h["stream"] == 3


        r=o["cassandra_response"]
        h2= r["response_headers"]
        assert h2["version"] == "4"
        assert h2["length"] == 107
        assert h2["op"] == "RESULT"
        assert h2["flags"] == "FLAG_0"
        assert h2["stream"] == 3

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
            cassandra_compressor= "snappy",
        )

        self.run_packetbeat(pcap="cassandra/v4/cassandra_compressed.pcap",debug_selectors=["*"])
        objs = self.read_output()

        o = objs[0]
        print o
        assert o["type"] == "cassandra"
        assert o["port"] == 9042
        assert o["bytes_in"] == 52
        assert o["bytes_out"] == 10

        q=o["cassandra_request"]
        h= q["request_headers"]
        assert h["version"] == "4"
        assert h["op"] == "STARTUP"
        assert h["length"] == 43
        assert h["flags"] == "FLAG_0"
        assert h["stream"] == 0


        r=o["cassandra_response"]
        h2= r["response_headers"]
        assert h2["version"] == "4"
        assert h2["length"] == 1
        assert h2["op"] == "READY"
        assert h2["flags"] == "Compress"
        assert h2["stream"] == 0

        o = objs[1]
        print o
        assert o["type"] == "cassandra"
        assert o["port"] == 9042
        assert o["bytes_in"] == 53
        assert o["bytes_out"] == 10

        q=o["cassandra_request"]
        h= q["request_headers"]
        assert h["version"] == "4"
        assert h["op"] == "REGISTER"
        assert h["length"] == 44
        assert h["flags"] == "Compress"
        assert h["stream"] == 64


        r=o["cassandra_response"]
        h2= r["response_headers"]
        assert h2["version"] == "4"
        assert h2["length"] == 1
        assert h2["op"] == "READY"
        assert h2["flags"] == "Compress"
        assert h2["stream"] == 64

        o = objs[2]
        print o
        assert o["type"] == "cassandra"
        assert o["port"] == 9042
        assert o["bytes_in"] == 62
        assert o["bytes_out"] == 165

        q=o["cassandra_request"]
        assert q["query"] == "SELECT * FROM system.local WHERE key='local'"
        h= q["request_headers"]
        assert h["version"] == "4"
        assert h["op"] == "QUERY"
        assert h["length"] == 53
        assert h["flags"] == "Compress"
        assert h["stream"] == 0


        r=o["cassandra_response"]
        h2= r["response_headers"]
        assert h2["version"] == "4"
        assert h2["length"] == 156
        assert h2["op"] == "RESULT"
        assert h2["flags"] == "Compress"
        assert h2["stream"] == 64
        assert r["result_type"] == "rows"
        rows=r["rows"]
        assert rows["num_rows"] == 290917
        meta=rows["meta"]
        assert meta["col_count"] == 9
        assert meta["flags"] == "GlobalTableSpec"
        assert meta["keyspace"] == "system"
        assert meta["table"] == "peers"

