from base import BaseTest


class Test(BaseTest):

    def test_index_modified(self):
        """
        Test that beat stops in case elasticsearch index is modified and pattern not
        """
        self.render_config_template(
            elasticsearch={"index": "test"},
        )

        exit_code = self.run_beat()

        assert exit_code == 1
        assert self.log_contains(
            "setup.template.name and setup.template.pattern have to be set if index name is modified.") is True

    def test_index_not_modified(self):
        """
        Test that beat starts running if elasticsearch output is set
        """
        self.render_config_template(
            elasticsearch={"hosts": "localhost:9200"},
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        proc.check_kill_and_wait()

    def test_index_modified_no_pattern(self):
        """
        Test that beat stops in case elasticsearch index is modified and pattern not
        """
        self.render_config_template(
            elasticsearch={"index": "test"},
            es_template_name="test",
        )

        exit_code = self.run_beat()

        assert exit_code == 1
        assert self.log_contains(
            "setup.template.name and setup.template.pattern have to be set if index name is modified.") is True

    def test_index_modified_no_name(self):
        """
        Test that beat stops in case elasticsearch index is modified and name not
        """
        self.render_config_template(
            elasticsearch={"index": "test"},
            es_template_pattern="test",
        )

        exit_code = self.run_beat()

        assert exit_code == 1
        assert self.log_contains(
            "setup.template.name and setup.template.pattern have to be set if index name is modified.") is True

    def test_index_with_pattern_name(self):
        """
        Test that beat starts running if elasticsearch output with modified index and pattern and name are set
        """
        self.render_config_template(
            elasticsearch={"hosts": "localhost:9200"},
            es_template_name="test",
            es_template_pattern="test-*",
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        proc.check_kill_and_wait()
