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

#import "EditorWindow.h"

@implementation EditorWindow

- (id) initWithBeat:(NSString*)name config:(NSString*)path {
    if (self = [super initWithWindowNibName:@"EditorWindow"]) {
        self->beatName = name;
        self->filePath = path;
    }
    return self;
}

- (void)windowDidLoad {
    [super windowDidLoad];
    verticalStackView.translatesAutoresizingMaskIntoConstraints = YES;
    [[self window] setTitle:[NSString stringWithFormat:@"%@ configuration", beatName]];

    NSError *err = nil;
    sourceText = [NSString stringWithContentsOfFile:filePath encoding:NSUTF8StringEncoding error:&err];
    if (sourceText == nil) {
        sourceText = [err localizedDescription];
    }
    NSTextStorage *storage = [(NSTextView*)[textEditor documentView] textStorage];
    [[storage mutableString] setString:sourceText];
    // Yaml needs a monospace font
    [storage setFont:[NSFont userFixedPitchFontOfSize:-1]];
}

- (BOOL)onClose
{
    NSTextStorage *storage = [(NSTextView*)[textEditor documentView] textStorage];
    if (![[storage string] isEqualToString:sourceText]) {
        NSAlert *alert = [[NSAlert alloc] init];
        [alert addButtonWithTitle:@"Discard"];
        [alert addButtonWithTitle:@"Continue editing"];
        [alert setMessageText:@"Discard changes?"];
        [alert setInformativeText:@"Changes will be lost if the dialog is closed without saving."];
        [alert setAlertStyle:NSAlertStyleWarning];
        if ([alert runModal] != NSAlertFirstButtonReturn) {
            return NO;
        }
    }
    [NSApp stopModalWithCode:NSModalResponseStop];
    return YES;
}

- (IBAction)saveAndCloseTapped:(id)sender
{
    NSError *err = nil;
    NSTextStorage *storage = [(NSTextView*)[textEditor documentView] textStorage];
    if (![[storage string] writeToFile:filePath atomically:YES encoding:NSUTF8StringEncoding error:&err]) {
        NSAlert *alert = [NSAlert alertWithError:err];
        [alert runModal];
        return;
    }
    [NSApp stopModalWithCode:NSModalResponseOK];
    [self close];
}

- (IBAction)closeTapped:(id)sender
{
    if ([self onClose]) {
        [NSApp stopModalWithCode:NSModalResponseStop];
        [self close];
    }
}

- (BOOL)windowShouldClose:(id)sender {
    return [self onClose];
}

@end
