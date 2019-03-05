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

#import <Foundation/Foundation.h>
#import "../Authorization.h"

@protocol Beat
- (bool) isRunning;
- (bool) isBoot;
- (int) pid;
- (NSString*) name;
- (NSString*) displayName;
- (NSString*) plistPath;
- (NSString*) configFile;
- (NSString*) logsPath;
- (BOOL) startWithAuth:(id<AuthorizationProvider>)auth;
- (BOOL) stopWithAuth:(id<AuthorizationProvider>)auth;
- (BOOL) toggleRunAtBootWithAuth:(id<AuthorizationProvider>)auth;
- (BOOL) uninstall;
@end

@protocol Beats
- (NSArray*) listBeats;
- (id <Beat>)getBeat:(NSString*)name;
@end
