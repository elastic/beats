// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package common

// This is how many data we dump from sk_buff->data to read full packet headers
// (IP + UDP header). This has been observed to include up to 100 bytes of
// padding.
const SkBuffDataDumpBytes = 256

// Fetching data from execve is complicated as support for strings or arrays
// in Kprobes appeared in recent kernels (~2018). To be compatible with older
// kernels it needs to dump fixed-size arrays in 8-byte chunks. As the total
// number of fetchargs available is limited, we have to dump only the first
// 128 bytes of every argument.
const MaxProgArgLen = 128
const MaxProgArgs = 5
