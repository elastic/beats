//
//  common.m
//  beats-preference-pane
//
//  Created by Adrian Serrano on 21/02/2018.
//  Copyright © 2018 Elastic. All rights reserved.
//

#import "common.h"
#import <Cocoa/Cocoa.h>
#import <sys/time.h>

static void readLines(NSFileHandle *handle, BOOL (^callback)(NSString*)) {
    const int readLength = 4096;
    NSMutableData *buffer = [NSMutableData dataWithCapacity:readLength];

    unsigned int length = 0;
    for (NSData *readData; (readData = [handle readDataOfLength:readLength])!= nil && [readData length] > 0;) {
        [buffer appendData:readData];
        unsigned int start = 0, // where the first line starts
                     base = length; // where it begins scan for newlines
        length += [readData length];
        char *bytes = [buffer mutableBytes];
        for (unsigned int i=base; i < length; i++) {
            if (bytes[i] == '\n') {
                NSString *line = [[NSString alloc] initWithBytesNoCopy:&bytes[start]
                                                                length:(i - start) encoding:NSUTF8StringEncoding
                                                          freeWhenDone:NO];
                callback(line);
                start = i + 1;
            }
        }
        // discard full lines
        if (start != 0) {
            [buffer replaceBytesInRange:NSMakeRange(0, start) withBytes:NULL length:0];
            length -= start;
        }
    }
}

int executeAndGetOutput(NSString *path, NSArray* args, BOOL (^callback)(NSString*)) {
    NSPipe *pipe = [NSPipe pipe];
    NSFileHandle *fHandle = pipe.fileHandleForReading;
    NSTask *task = [[NSTask alloc] init];
    task.launchPath = path;
    task.arguments = args;
    task.standardOutput = pipe;

    [task launch];

    readLines(fHandle, callback);

    [fHandle closeFile];
    [task waitUntilExit];
    return [task terminationStatus];
}

uint64_t getTimeMicroseconds(void) {
    struct timeval tv;
    gettimeofday(&tv, NULL);
    return tv.tv_sec*1000000 + tv.tv_usec;
}

NSString *strOrNil(NSString *str) {
    return str != nil? str : @"(nil)";
}
