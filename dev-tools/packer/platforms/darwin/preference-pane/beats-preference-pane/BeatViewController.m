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

#import "BeatViewController.h"
#import "EditorWindow.h"
#import "common/common.h"

@implementation BeatViewController

- (id) initWithBeat:(id<Beat>)beat
               auth:(id<AuthorizationProvider>)auth
             bundle:(NSBundle*)bundle
     beatsInterface:(id<Beats>)beatsInterface;
{
    if (self = [self initWithNibName:@"BeatView" bundle:bundle]) {
        self->beat = beat;
        self->auth = auth;
        self->beatsInterface = beatsInterface;
    }
    return self;
}

- (void)viewDidLoad {
    [super viewDidLoad];
    [self updateUI];
}

- (void)updateUI {
    id<Beat> beat = self->beat;

    if ([beat isRunning]) {
        [statusLabel setStringValue:[NSString stringWithFormat:@"%@ is running with PID %d", [beat displayName], [beat pid]]];
        [startStopButton setTitle:@"Stop"];
    } else {
        [statusLabel setStringValue:[NSString stringWithFormat:@"%@ is stopped", [beat displayName]]];
        [startStopButton setTitle:@"Start"];
    }

    if ([beat isBoot]) {
        [bootLabel setStringValue:@"Automatic start at boot is enabled"];
        [bootButton setTitle:@"Disable"];
    } else {
        [bootLabel setStringValue:@"Automatic start at boot is disabled"];
        [bootButton setTitle:@"Enable"];
    }
    [configField setStringValue:strOrNil([beat configFile])];
    [logsField setStringValue:strOrNil([beat logsPath])];

    BOOL unlocked = [auth isUnlocked];
    [startStopButton setEnabled:unlocked];
    [bootButton setEnabled:unlocked];
    [editButton setEnabled:unlocked];
    [logsButton setEnabled:unlocked];
}

- (void) update {
    beat = [beatsInterface getBeat:[beat name]];
    [self updateUI];
}

- (IBAction)startStopTapped:(id)sender {
    if (![auth isUnlocked]) {
        return;
    }
    uint64_t took = getTimeMicroseconds();
    id<Beat> beat = self->beat;

    if ([beat isRunning]) {
        [beat stopWithAuth:auth];
    } else {
        [beat startWithAuth:auth];
    }
    took = getTimeMicroseconds() - took;
    NSLog(@"start/stop took %lld us", took);
    [self update];
}

- (IBAction)startAtBootTapped:(id)sender {
    if (![auth isUnlocked]) {
        return;
    }
    [beat toggleRunAtBootWithAuth:auth];
    [self update];
}

- (IBAction)editConfigTapped:(id)sender {
    if (![auth isUnlocked]) {
        return;
    }
    id<Beat> beat = self->beat;
    NSString *conf = [beat configFile];

    // Create a temporal file with current user permissions
    NSString *tmpFile = [NSString stringWithFormat:@"%@/beatconf-%@.yml",NSTemporaryDirectory(), [[NSUUID UUID] UUIDString]];
    [@"" writeToFile:tmpFile atomically:NO encoding:NSUTF8StringEncoding error:nil];

    // Cat the config file contents into the temporal file
    [auth runAsRoot:@"/bin/sh" args:@[@"-c", [NSString stringWithFormat:@"cat '%@' > '%@'", conf, tmpFile]]];

    // Display editor on temp file
    EditorWindow *editor = [[EditorWindow alloc] initWithBeat:[beat displayName] config:tmpFile];
    NSWindow *window = [editor window];
    [window setFrameOrigin:[[[self view] window] frame].origin];
    NSModalResponse resp = [NSApp runModalForWindow: window];

    if (resp == NSModalResponseOK) {
        // Cat temporal file contents into config file.
        while ([auth runAsRoot:@"/bin/sh" args:@[@"-c", [NSString stringWithFormat:@"cat '%@' > '%@'", tmpFile, conf]]] != errAuthorizationSuccess) {
            // Authorization expired because the user took a while to edit the config
            // Ask to reauthorize
            NSAlert *alert = [[NSAlert alloc] init];
            [alert addButtonWithTitle:@"Retry"];
            [alert addButtonWithTitle:@"Cancel"];
            [alert setMessageText:@"Retry authentication?"];
            [alert setInformativeText:@"Authentication expired. Configuration changes will be lost unless valid credentials are provided."];
            [alert setAlertStyle:NSAlertStyleWarning];
            if ([alert runModal] != NSAlertFirstButtonReturn) {
                break;
            }
            [auth forceUnlock];
        }
    }

    [[NSFileManager defaultManager] removeItemAtPath:tmpFile error:nil];
}

- (IBAction)viewLogsTapped:(id)sender
{
    NSAlert *alert = [[NSAlert alloc] init];
    [alert addButtonWithTitle:@"OK"];
    [alert setMessageText:@"Can't display logs"];
    [alert setInformativeText:@"Due to strict permissions in Beats logs, they are only accessible using the command line as root."];
    [alert setAlertStyle:NSAlertStyleWarning];
    [alert runModal];
}

@end
