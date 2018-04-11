// Package redis package contains input and harvester to read the redis slow log
//
// The redis slow log is stored in memory. The slow log can be activate on the redis command line as following:
//
// 	CONFIG SET slowlog-log-slower-than 2000000
//
// This sets the value of the slow log to 2000000 micro seconds (2s). All queries taking longer will be reported.
//
// As the slow log is in memory, it can be configured how many items it consists:
//
// 	CONFIG SET slowlog-max-len 200
//
// This sets the size of the slow log to 200 entries. In case the slow log is full, older entries are dropped.
package redis
