//
//  common.h
//  beats-preference-pane
//
//  Created by Adrian Serrano on 21/02/2018.
//  Copyright © 2018 Elastic. All rights reserved.
//

#import <Foundation/Foundation.h>

// executes the given `path` executable, passing `args` array.
// Callback is called for every line in the program's output.
// Returns the program exit status.
int executeAndGetOutput(NSString *path, NSArray *args, BOOL (^callback)(NSString*));

// Returns the current time in microseconds
uint64_t getTimeMicroseconds(void);

// Returns the given string, or @"nil" if its nil.
NSString *strOrNil(NSString *str);
