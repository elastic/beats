/*
Package info fetches Redis server information and statistics using the Redis
INFO command.

The current implementation is tested with redis 3.2.3
More details on all the fields provided by the redis info command can be found here: http://redis.io/commands/INFO

`info.go` uses the Redis `INFO default` command for stats. This allows us to fetch  all metrics at once and filter out
undesired metrics based on user configuration on the client. The alternative would be to fetch each type as an
independent `INFO` call, which has the potential of introducing higher latency (e.g., more round trip Redis calls).

*/
package info
