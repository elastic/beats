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

#import "TabViewDelegate.h"
#import "BeatViewController.h"
#import "common/common.h"

@implementation TabViewDelegate
- (id) initWithTabView:(NSTabView *)tabView
                bundle:(NSBundle*)bundle
                 beats:(id<Beats>)beats
{
    if (self = [super init]) {
        self->selectedTab = nil;
        self->tabView = tabView;
        self->bundle = bundle;
        self->beatsInterface = beats;
        tabView.delegate = self;
    }
    return self;
}

- (void) update
{
    [selectedTab update];
}

- (void) populateTabs:(NSArray*)beats withAuth:(id<AuthorizationProvider>)auth
{
    // cache self->selectedTab, as it is going to change in this method
    // (add|remove|select)TabViewItem methods call the NSTabViewDelegate callbacks
    BeatViewController *selectedTab = self->selectedTab;
    uint i;
    NSArray *items;
    NSString *selectedName = nil;
    for (i=0, items = tabView.tabViewItems; items != nil && i < items.count; i++) {
        NSTabViewItem *item = [items objectAtIndex:i];
        if (selectedTab != nil && item.viewController == selectedTab) {
            selectedName = item.identifier;
        }
        [tabView removeTabViewItem:item];
    }
    for (uint i=0; i < beats.count; i++) {
        NSString *beatName = [beats objectAtIndex:i];
        id<Beat> beat = [beatsInterface getBeat:beatName];
        if (beat == nil) {
            // TODO: Investigate and repair. Why some beats seem to break. Seemingly after some time disabled
            //       they are unloaded from launchctl.
            NSLog(@"Ignoring broken beat %@", beatName);
            continue;
        }
        NSTabViewItem *item = [[NSTabViewItem alloc] initWithIdentifier:beatName];
        [item setLabel:[beat displayName]];
        BeatViewController *vc = [[BeatViewController alloc]
                                 initWithBeat:[beatsInterface getBeat:beatName] auth:auth bundle:bundle beatsInterface:beatsInterface];
        [item setViewController:vc];
        [tabView addTabViewItem:item];
        if ([beatName isEqualToString:selectedName]) {
            selectedTab = vc;
            [tabView selectTabViewItem:item];
        }
    }
}

- (void) tabViewDidChangeNumberOfTabViewItems:(NSTabView*) tabView
{
    // ignore
}

- (BOOL) tabView:(NSTabView*)tabView shouldSelectTabViewItem:(NSTabViewItem*)item
{
    return YES;
}

- (void) tabView:(NSTabView*)tabView willSelectTabViewItem:(NSTabViewItem*)item
{
    [(BeatViewController*)[item viewController] update];
}

- (void) tabView:(NSTabView*)tabView didSelectTabViewItem:(NSTabViewItem*)item
{
    selectedTab = (BeatViewController*)[item viewController];
}

@end
