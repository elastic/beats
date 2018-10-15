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

/*
Package eventlogging provides access to the Event Logging API that was designed
for applications that run on the Windows Server 2003, Windows XP, or Windows
2000 operating system.

It can be used on new versions of Windows (i.e. Windows Vista, Windows 7,
Windows Server 2008, Windows Server 2012), but the preferred API for those
systems is the Windows Event Log API. See the wineventlog package.
*/
package eventlogging
