// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package mage

// "github.com/elastic/beats/v7/x-pack/osquerybeat/internal/distro"

// CustomizePackaging modifies the package specs to add the modules and
// modules.d directory. You must declare a dependency on either
// PrepareModulePackagingOSS or PrepareModulePackagingXPack.

// func CustomizePackaging() {
// 	for _, args := range devtools.Packages {
// 		distFile := distro.OsquerydDistroPlatformFilename(args.OS)

// 		// The minimal change to fix the issue for 7.13
// 		// https://github.com/elastic/beats/issues/25762
// 		// TODO: this could be moved to dev-tools/packaging/packages.yml for the next release
// 		var mode os.FileMode = 0644
// 		// If distFile is osqueryd binary then it should be executable
// 		if distFile == distro.OsquerydFilename() {
// 			mode = 0750
// 		}
// 		arch := defaultArch
// 		if args.Arch != "" {
// 			arch = args.Arch
// 		}
// 		packFile := devtools.PackageFile{
// 			Mode:   mode,
// 			Source: filepath.Join(distro.GetDataInstallDir(distro.OSArch{OS: args.OS, Arch: arch}), distFile),
// 		}
// 		args.Spec.Files[distFile] = packFile
// 	}
// }

// // Todo cloudbeat write mage script to package beat with agent
