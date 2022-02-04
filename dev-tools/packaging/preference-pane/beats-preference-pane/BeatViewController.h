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

#import <Cocoa/Cocoa.h>

#import "beats/Beats.h"
#import "Authorization.h"

// BeatViewController handles the individual UI view for a beat
@interface BeatViewController : NSViewController
{
    IBOutlet NSTextField *statusLabel;
    IBOutlet NSTextField *bootLabel;
    IBOutlet NSTextField *configField;
    IBOutlet NSTextField *logsField;
    IBOutlet NSButton *startStopButton;
    IBOutlet NSButton *bootButton;
    IBOutlet NSButton *editButton;
    IBOutlet NSButton *logsButton;

    // The Beat being displayed by this view
    id<Beat> beat;
    id<Beats> beatsInterface;
    id<AuthorizationProvider> auth;
}

- (id)initWithBeat:(id<Beat>)_ auth:(id<AuthorizationProvider>)_ bundle:(NSBundle*)_ beatsInterface:(id<Beats>)_;
- (IBAction)startStopTapped:(id)sender;
- (IBAction)startAtBootTapped:(id)sender;
- (IBAction)editConfigTapped:(id)sender;
- (IBAction)viewLogsTapped:(id)sender;
- (void)update;

@end
