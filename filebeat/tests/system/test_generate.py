import os
import shutil
import filebeat


class Test(filebeat.BaseTest):
    def test_generate_module(self):
        """
        Test filebeat generate module my_module generates a new module
        """

        self._create_clean_test_modules()
        exit_code = self.run_beat(
            extra_args=["generate", "module", "my_module", "-modules-path", "test_modules", "-es-beats", self.beat_path])
        assert exit_code == 0

        module_root = os.path.join("test_modules", "module", "my_module")
        module_meta_root = os.path.join(module_root, "_meta")
        self._assert_required_module_directories_are_created(module_root, module_meta_root)
        self._assert_required_module_files_are_created_and_substitution_is_done(module_root, module_meta_root)

        shutil.rmtree("test_modules")

    def _assert_required_module_directories_are_created(self, module_root, module_meta_root):
        expected_created_directories = [
            module_root,
            module_meta_root,
        ]
        for expected_dir in expected_created_directories:
            assert os.path.isdir(expected_dir)

    def _assert_required_module_files_are_created_and_substitution_is_done(self, module_root, module_meta_root):
        expected_created_template_files = [
            os.path.join(module_root, "module.yml"),
            os.path.join(module_meta_root, "config.yml"),
            os.path.join(module_meta_root, "docs.asciidoc"),
            os.path.join(module_meta_root, "fields.yml"),
        ]
        for template_file in expected_created_template_files:
            assert os.path.isfile(template_file)
            assert '{module}' not in open(template_file)

    def test_generate_fileset(self):
        """
        Test filebeat generate fileset my_module my_fileset generates a new fileset
        """

        self._create_clean_test_modules()
        exit_code = self.run_beat(
            extra_args=["generate", "module", "my_module", "-modules-path", "test_modules", "-es-beats", self.beat_path])
        assert exit_code == 0

        exit_code = self.run_beat(
            extra_args=["generate", "fileset", "my_module", "my_fileset", "-modules-path", "test_modules", "-es-beats", self.beat_path])
        assert exit_code == 0

        fileset_root = os.path.join("test_modules", "module", "my_module", "my_fileset")
        self._assert_required_fileset_directories_are_created(fileset_root)
        self._assert_required_fileset_files_are_created_and_substitution_is_done(fileset_root)

        shutil.rmtree("test_modules")

    def _assert_required_fileset_directories_are_created(self, fileset_root):
        expected_created_directories = [
            fileset_root,
            os.path.join(fileset_root, "config"),
            os.path.join(fileset_root, "ingest"),
            os.path.join(fileset_root, "_meta"),
            os.path.join(fileset_root, "test"),
        ]
        for expected_dir in expected_created_directories:
            assert os.path.isdir(expected_dir)

    def _assert_required_fileset_files_are_created_and_substitution_is_done(self, fileset_root):
        expected_created_template_files = [
            os.path.join(fileset_root, "config", "my_fileset.yml"),
            os.path.join(fileset_root, "ingest", "pipeline.json"),
            os.path.join(fileset_root, "manifest.yml"),
        ]
        for template_file in expected_created_template_files:
            assert os.path.isfile(template_file)
            assert '{fileset}' not in open(template_file)

    def test_generate_fields_yml(self):
        """
        Test filebeat generate fields my_module my_fileset generates a new fields.yml for my_module/my_fileset
        """

        self._create_clean_test_modules()
        exit_code = self.run_beat(
            extra_args=["generate", "module", "my_module", "-modules-path", "test_modules", "-es-beats", self.beat_path])
        assert exit_code == 0

        exit_code = self.run_beat(
            extra_args=["generate", "fileset", "my_module", "my_fileset", "-modules-path", "test_modules", "-es-beats", self.beat_path])
        assert exit_code == 0

        test_pipeline_path = os.path.join(self.beat_path, "tests", "system", "input", "my-module-pipeline.json")
        fileset_pipeline = os.path.join("test_modules", "module",
                                        "my_module", "my_fileset", "ingest", "pipeline.json")

        print(os.path.isdir("test_modules"))
        print(os.path.isdir(os.path.join("test_modules", "module")))
        print(os.path.isdir(os.path.join("test_modules", "module", "my_module")))
        print(os.path.isdir(os.path.join("test_modules", "module", "my_module", "my_fileset")))
        print(os.path.isdir(os.path.join("test_modules", "module", "my_module", "my_fileset", "ingest")))
        print(os.path.isdir(os.path.join("test_modules", "module", "my_module", "my_fileset", "ingest")))
        print(os.path.exists(os.path.join("test_modules", "module", "my_module", "my_fileset", "ingest", "pipeline.json")))
        print(os.path.exists(fileset_pipeline))
        print(os.path.exists(test_pipeline_path))
        shutil.copyfile(test_pipeline_path, fileset_pipeline)

        print(fileset_pipeline)
        print(os.path.abspath(fileset_pipeline))
        exit_code = self.run_beat(
            extra_args=["generate", "fields", "my_module", "my_fileset", "-es-beats", "test_modules", "-without-documentation"])
        assert exit_code == 0

        fields_yml_path = os.path.join("test_modules", "module", "my_module", "my_fileset", "_meta", "fields.yml")
        assert os.path.isfile(fields_yml_path)

        shutil.rmtree("test_modules")

    def _create_clean_test_modules(self):
        if os.path.isdir("test_modules"):
            shutil.rmtree("test_modules")
        os.mkdir("test_modules")
