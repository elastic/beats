//
// Copyright (c) 2012–2018 Elastic <http://www.elastic.co>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

// Service prefix used by Beats launch daemons. Used for detection
#define BEATS_PREFIX @"co.elastic.beats"

// How often daemons info is updated
#define UPDATE_INTERVAL_SECS 2.0

// Helper binary name
#define HELPER_BINARY @"helper"

// Path where to look for launch services
#define LAUNCHDAEMONS_PATH @"/Library/LaunchDaemons"

// Path to launchctl executable
#define LAUNCHCTL_PATH @"/bin/launchctl"
