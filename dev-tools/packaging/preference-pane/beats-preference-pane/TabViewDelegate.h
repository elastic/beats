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
#import <Cocoa/Cocoa.h>

#import "beats/Beats.h"
#import "Authorization.h"

@class BeatViewController;

/* TabViewDelegate takes care of the NSTabView that displays all the installed beats
 */
@interface TabViewDelegate : NSObject <NSTabViewDelegate> {
    NSTabView *tabView;
    NSBundle *bundle;
    BeatViewController *selectedTab;
    id <Beats> beatsInterface;
}
- (id) initWithTabView:(NSTabView*)_ bundle:(NSBundle*)_ beats:(id<Beats>)_;
- (void) update;
- (void) populateTabs:(NSArray*)_ withAuth:(id<AuthorizationProvider>)_;

// NSTabViewDelegate
- (void) tabViewDidChangeNumberOfTabViewItems:(NSTabView*)_;
- (BOOL) tabView:(NSTabView*)_ shouldSelectTabViewItem:(NSTabViewItem*)_;
- (void) tabView:(NSTabView*)_ willSelectTabViewItem:(NSTabViewItem*)_;
- (void) tabView:(NSTabView*)_ didSelectTabViewItem:(NSTabViewItem*)_;
@end
