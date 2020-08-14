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

static void fail(NSString *msg) {
    fprintf(stderr, "%s\n", [msg cStringUsingEncoding:NSUTF8StringEncoding]);
}

// setRunAtBoot loads a property list for a launch daemon,
// changes the value of the RunAtLoad property, and writes it
// down to disk again.
BOOL setRunAtBoot(NSString* plistPath, BOOL runAtBoot) {
    // Mutable property list so it can be changed in-place
    NSPropertyListMutabilityOptions opts = NSPropertyListMutableContainersAndLeaves;
    NSPropertyListFormat format = 0;
    NSError *err = nil;
    NSInputStream *input = [[NSInputStream alloc] initWithFileAtPath:plistPath];
    if (input == nil) {
        fail(@"Unable to open input file");
        return NO;
    }
    [input open];
    err = [input streamError];
    if (err != nil) {
        fail([NSString stringWithFormat:@"Unable to open input stream. Code=%u `%@`", (unsigned int)[err code], [err localizedDescription]]);
        return NO;
    }
    
    NSMutableDictionary *dict = [NSPropertyListSerialization
                                 propertyListWithStream:input
                                 options:opts
                                 format:&format
                                 error:&err];
    if (err != nil) {
        fail([NSString stringWithFormat:@"Error reading property list. Code=%u `%@`", (unsigned int)[err code], [err localizedDescription]]);
        return NO;
    }
    [input close];
    NSNumber *curValue = dict[@"RunAtLoad"];
    if (curValue != nil && [curValue boolValue] == runAtBoot) {
        fail(@"RunAtLoad setting already has required value");
        return YES;
    }
    NSNumber *newValue = [NSNumber numberWithBool:runAtBoot];
    [dict setValue:newValue forKey:@"RunAtLoad"];

    NSOutputStream *output = [NSOutputStream outputStreamToMemory];
    [output open];
    err = [output streamError];
    if (err != nil) {
        fail([NSString stringWithFormat:@"Error creating stream. Code=%u `%@`", (unsigned int)[err code], [err localizedDescription]]);
        return NO;
    }
    
    [NSPropertyListSerialization writePropertyList:dict
                                          toStream:output
                                            format:format
                                           options:0
                                             error:&err];
    if (err == nil) {
        err = [output streamError];
    }
    if (err != nil) {
        fail([NSString stringWithFormat:@"Error writing property-list. Code=%u `%@`", (unsigned int)[err code], [err localizedDescription]]);
        return NO;
    }
    [output close];
    
    NSData *data = [output propertyForKey:NSStreamDataWrittenToMemoryStreamKey];
    BOOL success = [data writeToFile:plistPath atomically:YES];
    if (!success) {
        fail(@"Error overwriting plist file");
        return NO;
    }
    return YES;
}
