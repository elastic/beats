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
// Input Loader with registry and plugins
//
// TODO: document
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
//
// Self contained InputManager
//
// TODO: document
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
//
package v2
