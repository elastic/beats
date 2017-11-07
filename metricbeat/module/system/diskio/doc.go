/*
Package diskio fetches disk IO metrics from the OS. It is implemented for
darwin (requires cgo), freebsd, linux, and windows.

Detailed descriptions of IO stats provided by Linux can be found here:
https://git.kernel.org/cgit/linux/kernel/git/torvalds/linux.git/plain/Documentation/iostats.txt?id=refs/tags/v4.6-rc7
*/
package diskio
