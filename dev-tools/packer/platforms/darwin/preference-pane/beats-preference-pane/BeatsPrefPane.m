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

#import "config.h"
#import "BeatsPrefPane.h"
#import "beats/BeatsService.h"

@implementation BeatsPrefPane

// Constructor
- (id)initWithBundle:(NSBundle *)bundle
{
    if ( ( self = [super initWithBundle:bundle] ) != nil ) {
        self->beatsInterface = [[BeatsService alloc] initWithPrefix:BEATS_PREFIX];
        self->updateTimer = nil;
        self->knownBeats = [beatsInterface listBeats];
        self->bundle = bundle;
        self->helperPath = [bundle pathForAuxiliaryExecutable:HELPER_BINARY];
        NSLog(@"Using helper: `%@`", helperPath);
    }
    return self;
}

// Called when UI file is loaded
- (void)mainViewDidLoad
{
    // Setup SFAuthorizationView
    AuthorizationItem items = {kAuthorizationRightExecute, 0, NULL, 0};
    AuthorizationRights rights = {1, &items};
    [authView setAuthorizationRights:&rights];
    authView.delegate = self;
    [authView updateStatus:nil];
    // Allocate tabview delegate
    tabDelegate = [[TabViewDelegate alloc] initWithTabView:beatsTab bundle:bundle beats:beatsInterface];
}

// Called before the preference pane is shown
- (void)willSelect
{
    [self updateUI];
}

// Called when the preference pane is shown
- (void)didSelect
{
    updateTimer = [NSTimer scheduledTimerWithTimeInterval:UPDATE_INTERVAL_SECS target:self selector:@selector(onTimer) userInfo:nil repeats:YES];
}

// Called when the preference pane is closed
- (void)didUnselect
{
    [updateTimer invalidate];
    updateTimer = nil;
}

// Custom code to update the UI elements
- (void)updateUI {
    [tabDelegate populateTabs:knownBeats withAuth:self];
    [messageLabel setHidden:knownBeats.count > 0];
}

static BOOL beatArrayEquals(NSArray *a, NSArray *b)
{
    size_t n = a.count;
    if (b.count != n) return NO;
    for (size_t i = 0; i < n; i++) {
        if (![(NSString*)a[i] isEqualToString:b[i]])
            return NO;
    }
    return YES;
}

- (void)onTimer
{
    [authView updateStatus:nil];
    NSArray *beats = [beatsInterface listBeats];
    if (!beatArrayEquals(beats, knownBeats)) {
        knownBeats = beats;
        [self updateUI];
    } else {
        [tabDelegate update];
    }
}

//
// SFAuthorization delegates
//

- (void)authorizationViewDidAuthorize:(SFAuthorizationView *)view {
    // Update the tab delegate so that it can enable UI elements
    [tabDelegate update];
}

- (void)authorizationViewDidDeauthorize:(SFAuthorizationView *)view {
    // Update the tab delegate so that it can disable UI elements
    [tabDelegate update];
}

//
// AuthorizationProvider protocol
//

- (BOOL)isUnlocked {
    return [authView authorizationState] == SFAuthorizationViewUnlockedState;
}

- (int)runAsRoot:(NSString*)program args:(NSArray*)args {
    size_t numArgs = args.count;
    char **cArgs = alloca(sizeof(char*) * (1 + numArgs));
    for (int i=0; i<args.count; i++) {
        cArgs[i] = (char*)[(NSString*)[args objectAtIndex:i] cStringUsingEncoding:NSUTF8StringEncoding];
    }
    cArgs[numArgs] = NULL;

    NSLog(@"Running AuthorizationExecuteWithPrivileges(`%@ %@`)", program, [args componentsJoinedByString:@" "]);

    FILE *pipe = NULL;
    // TODO: AuthorizationExecuteWithPrivileges is deprecated. Migrate to SMJobBless
    int res = AuthorizationExecuteWithPrivileges([[authView authorization] authorizationRef],
                                       [program cStringUsingEncoding:NSUTF8StringEncoding],
                                       kAuthorizationFlagDefaults,
                                       cArgs,
                                       &pipe);
    if (res != errAuthorizationSuccess) {
        NSString *errMsg = (__bridge NSString*)SecCopyErrorMessageString(res, NULL);
        NSLog(@"Error: AuthorizationExecuteWithPrivileges(`%@ %@`) failed with error code %d: %@",
              program, [args componentsJoinedByString:@" "], res, errMsg);
        return res;
    }
    if (pipe != NULL) {
        const size_t bufLen = 1024;
        char buf[bufLen];
        while (fgets(buf, bufLen, pipe)) {
            NSLog(@"%@ output: %s", program, buf);
        }
        fclose(pipe);
    }
    return 0;
}

- (BOOL)forceUnlock {
    return [authView authorize:nil];
}

- (int)runHelperAsRootWithArgs:(NSArray *)args {
    return [self runAsRoot:helperPath args:args];
}

@end

