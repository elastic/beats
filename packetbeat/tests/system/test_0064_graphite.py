from packetbeat import BaseTest


class Test(BaseTest):
	"""
	Basic Graphite Tests
	"""

	def test_line_protocol(self):
		"""
		Should correctly pass for cases where the 
		input follows line protocol
		"""

		self.render_config_template(
			graphite_ports=[2003]
		)

		self.run_packetbeat(pcap="graphite_line_test.pcap")

		objs = self.read_output()

		assert all([o["type"] == "graphite" for o in objs])
		assert all([o["bytes_out"] == 0 for o in objs])

	def test_pickle_protocol(self):
		"""
		Should correctly pass for cases where the 
		input follows pickle protocol
		"""

		self.render_config_template(
			graphite_ports=[2003]
		)

		self.run_packetbeat(pcap="graphite_pickle_test.pcap")

		objs = self.read_output()

		assert all([o["type"] == "graphite" for o in objs])
		assert all([o["bytes_out"] == 0 for o in objs])



		


