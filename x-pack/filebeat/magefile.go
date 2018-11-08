// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build mage

package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/magefile/mage/mg"

	"github.com/elastic/beats/dev-tools/mage"
)

func init() {
	mage.BeatDescription = "Filebeat sends log files to Logstash or directly to Elasticsearch."
	mage.BeatLicense = "Elastic"
}

// Build builds the Beat binary.
func Build() error {
	return mage.Build(mage.DefaultBuildArgs())
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	return mage.GolangCrossBuild(mage.DefaultGolangCrossBuildArgs())
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return mage.CrossBuild()
}

// Fields generates a fields.yml and fields.go for each module.
func Fields() {
	mg.Deps(fieldsYML, mage.GenerateModuleFieldsGo)
}

// fieldsYML generates a fields.yml based on filebeat + x-pack/filebeat/modules.
func fieldsYML() error {
	return mage.GenerateFieldsYAML(mage.OSSBeatDir("module"), "module")
}

// Dashboards collects all the dashboards and generates index patterns.
func Dashboards() error {
	return mage.KibanaDashboards(mage.OSSBeatDir("module"), "module")
}

// Config generates both the short and reference configs.
func Config() {
	mg.Deps(shortConfig, referenceConfig, createDirModulesD)
}

// Update is an alias for running fields, dashboards, config.
func Update() {
	mg.SerialDeps(Fields, Dashboards, Config)
}

// -----------------------------------------------------------------------------
// Customizations specific to Filebeat.
// - Include modules directory in packages (minus _meta and test files).
// - Include modules.d directory in packages.

const (
	dirModuleGenerated   = "build/package/module"
	dirModulesDGenerated = "build/package/modules.d"
)

// prepareModulePackaging generates modules and modules.d directories
// for an x-pack distribution, excluding _meta and test files so that they are
// not included in packages.
func prepareModulePackaging() error {
	mg.Deps(createDirModulesD)

	err := mage.Clean([]string{
		dirModuleGenerated,
		dirModulesDGenerated,
	})
	if err != nil {
		return err
	}

	for _, copyAction := range []struct {
		src, dst string
	}{
		{mage.OSSBeatDir("module"), dirModuleGenerated},
		{"module", dirModuleGenerated},
		{mage.OSSBeatDir("modules.d"), dirModulesDGenerated},
		{"modules.d", dirModulesDGenerated},
	} {
		err := (&mage.CopyTask{
			Source:  copyAction.src,
			Dest:    copyAction.dst,
			Mode:    0644,
			DirMode: 0755,
			Exclude: []string{
				"/_meta",
				"/test",
				"fields.go",
			},
		}).Execute()
		if err != nil {
			return err
		}
	}
	return nil
}

func shortConfig() error {
	var configParts = []string{
		mage.OSSBeatDir("_meta/common.p1.yml"),
		mage.OSSBeatDir("_meta/common.p2.yml"),
		"{{ elastic_beats_dir }}/libbeat/_meta/config.yml",
	}

	for i, f := range configParts {
		configParts[i] = mage.MustExpand(f)
	}

	configFile := mage.BeatName + ".yml"
	mage.MustFileConcat(configFile, 0640, configParts...)
	mage.MustFindReplace(configFile, regexp.MustCompile("beatname"), mage.BeatName)
	mage.MustFindReplace(configFile, regexp.MustCompile("beat-index-prefix"), mage.BeatIndexPrefix)
	return nil
}

func referenceConfig() error {
	const modulesConfigYml = "build/config.modules.yml"
	err := mage.GenerateModuleReferenceConfig(modulesConfigYml, mage.OSSBeatDir("module"), "module")
	if err != nil {
		return err
	}
	defer os.Remove(modulesConfigYml)

	var configParts = []string{
		mage.OSSBeatDir("_meta/common.reference.p1.yml"),
		modulesConfigYml,
		mage.OSSBeatDir("_meta/common.reference.p2.yml"),
		"{{ elastic_beats_dir }}/libbeat/_meta/config.reference.yml",
	}

	for i, f := range configParts {
		configParts[i] = mage.MustExpand(f)
	}

	configFile := mage.BeatName + ".reference.yml"
	mage.MustFileConcat(configFile, 0640, configParts...)
	mage.MustFindReplace(configFile, regexp.MustCompile("beatname"), mage.BeatName)
	mage.MustFindReplace(configFile, regexp.MustCompile("beat-index-prefix"), mage.BeatIndexPrefix)
	return nil
}

func createDirModulesD() error {
	if err := os.RemoveAll("modules.d"); err != nil {
		return err
	}

	shortConfigs, err := filepath.Glob("module/*/_meta/config.yml")
	if err != nil {
		return err
	}

	for _, f := range shortConfigs {
		parts := strings.Split(filepath.ToSlash(f), "/")
		if len(parts) < 2 {
			continue
		}
		moduleName := parts[1]

		cp := mage.CopyTask{
			Source: f,
			Dest:   filepath.Join("modules.d", moduleName+".yml.disabled"),
			Mode:   0644,
		}
		if err = cp.Execute(); err != nil {
			return err
		}
	}
	return nil
}
