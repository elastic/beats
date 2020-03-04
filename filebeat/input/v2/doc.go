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

// Package v2 provides common interfaces and helper functions for defining an
// input extension point.
//
// Extension points need to provide support for querying and verifying its
// capabilities, as well as providing support for creating and running actual
// inputs. For this purpose the Registry and Plugin interfaces provide a common
// interface for describing capabilities. The Loader and Input interfaces
// provide support for creating and running the actual plugins.
//
// These interfaces are meant to be implemented by separate packages,
// potentially providing additional capabilities and services to the actual
// inputs. Having additional capabilities in dedicated packages ensures that
// the final binary only compiles in dependencies that are actually required
// for the selected set of Inputs and Extension point shall be able to run.
//
// This package provides additional helpers for combining the loaders and
// registries defined in different packages, to create the actual extension point.
// For example LoaderList and LoaderTable combine multiple Loaders into one
// single loader, and ConfigsLoader allows developers to create a loader that
// produces configurations for other configured Loaders (similar to filesets).
//
// A set of registries is combined using RegistryList. Use
// `RegistryList.Validate` to check if the combined registries do not provide
// similar input names.
//
//
// Self contained InputManager
//
// With this setup no actual Plugin or Input variants do exist.
//
//    +-------------+     +---------------------+     +------------------+      +-----------+
//    |  v2.Loader  | <.. |                     | --> | ext.InputManager | <--> | ext.input | <+
//    +-------------+     |                     |     +------------------+      +-----------+  |
//                        |                     |                                 ^            |
//                        | ext.InputConfigurer | ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~*            |
//                        |                     |                                              |
//    +-------------+     |                     |                                              |
//    | v2.Registry | <.. |                     |                                              |
//    +-------------+     +---------------------+                                              |
//                          }                                                                  |
//                          }                                                                  |
//                          v                                                                  |
//                        +---------------------+                                              |
//                        |      v2.Input       | ---------------------------------------------+
//                        +---------------------+
//
// The InputConfigurer implements the v2.Registry interface, to report the potential capabilities
// provided by this package. The Loader interface is used create a 'handle' that can be used to
// dynamically add/remove an workers to the InputManager, based on the users configuration.
//
//
// Input Loader with registry and plugins
//
// In this scenario the package centers around the Registry. Plugins are added
// to the registry, and create standalone input instances.
//
//                                                                 +-------------+
//                                                                 |  v2.Plugin  |
//                                                                 +-------------+
//                                                                   ^
//                                           +-------------+         :
//                                           v             |         :
//    +-----------+     +------------+     +-----------------+     +-------------+     +-----------+
//    | v2.Loader | <.. | ext.Loader | --> |                 | --> | ext.Plugin  | ~~> | ext.Input |
//    +-----------+     +------------+     |                 |     +-------------+     +-----------+
//                        }                |                 |
//                        }                |  ext.Registry   |
//                        v                |                 |
//                      +------------+     |                 |     +-------------+
//                      |  v2.Input  |     |                 | ..> | v2.Registry |
//                      +------------+     +-----------------+     +-------------+
//                                           H
//                                           H
//                                           v
//                                         +-----------------+
//                                         | v2.RegistryTree |
//                                         +-----------------+
//
// The input type is defined in the package directly and the Loader wraps the
// input into a v2.Input. The Loader will use the registry to create Inputs.
// Registry and Loader each implement the v2.Loader and v2.Registry interfaces
// and represent the extensions to be used with an extension point. The
// ext.Registry uses v2.RegistryTree to keep track of the Plugins and
// sub-registries.
//
// For example:
//    type Registry v2.RegistryTree
//
//    type Extension interface {
//        addToRegistry(*Registry) error
//    }
//
//    func (r *Registry) Each(fn func(v2.Plugin) bool) { (*v2.RegistryTree)(r).Each(fn) }
//    func (r *Registry) Find(name string) (plugin v2.Plugin, ok bool) { return (*v2.RegistryTree)(r).Find(name) }
//    func (r *Registry) Add(ext Extension) error { return ext.addToRegistry(r) }
//
//    func (r *Registry) addToRegistry(parent *Registry) error {
//        return (*v2.RegistryTree)(parent).AddRegistry(r)
//    }
//
//    func (p *Plugin) addToRegistry(parent *Registry) error {
//        return (*v2.RegistryTree)(parent).AddPlugin(p)
//    }
//
// The `Extension` interface allows users to combine registries from multiple sources/packages. For example:
//
//    reg := new(Registry
//    reg.Add(oss.SimpleInputs, xpack.SimpleInputs)
//
//
// Combination of InputManager with Plugins
//
// Variations of these two first two implementations might combine an input
// manager with pluggable input types. The InputManager could transparently
// handle all communication with the registry, while the Input creates events
// with corresponding type-safe state updated.
//
// For example:
//
//    type Input struct {
//        Run(ctx Context, publisher Publisher, cursor interface{}) error
//    }
//
//    type Publisher interface {
//        Publish(evt beat.Event, cursor interface{}) error
//    }
//
// The InputManager will pass the cursor (last known state) to the input to
// continue processing. Each event to be published must be accomodated with a
// new cursor state.
//
// The InputManager will check that no more than 2 inputs of the same type
// collect the same data (exclusive access), and that state is correctly
// synchronised after ACK with the registry file.
//
package v2
