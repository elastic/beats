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

#import <PreferencePanes/PreferencePanes.h>
#import <SecurityInterface/SFAuthorizationView.h>

#import "TabViewDelegate.h"
#import "Authorization.h"

/* BeatsPrefPane is the main class for handling the preference pane.
   Implements <AuthorizationProvided> so that it can provide authorization
   obtained via the SFAuthorizationView to other components.
 */
@interface BeatsPrefPane : NSPreferencePane <AuthorizationProvider> {
    IBOutlet NSTabView *beatsTab;
    IBOutlet TabViewDelegate *tabDelegate;
    IBOutlet SFAuthorizationView *authView;
    IBOutlet NSTextField *messageLabel;
    NSTimer *updateTimer;
    NSBundle *bundle;
    NSArray *knownBeats;
    NSString *helperPath;
    id<Beats> beatsInterface;
}

- (id)initWithBundle:(NSBundle *)bundle;
- (void)mainViewDidLoad;
- (void)didSelect;
- (void)willSelect;
- (void)didUnselect;
- (BOOL)isUnlocked;
@end
