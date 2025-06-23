# Competitor's comparison

In summary Vector, OTel, Splunk can ingest gzipped files.
- **Vector and Splunk** decompress them for reading. Vector does not explain what it mean by that, but Splunk does: "decompress archive files [...] prior to processing"
- Vector auto detects them, does not rely on extension/naming convention. Splunk seems to use the file name for detection, OTel filelog receiver needs a config
- none keep track of the ingest offset. Vector never re-ingests once it reads a file, regardless if it reads until the end or not. Splunk re-ingests the whole file if new data is added. However as it says it decompress the files first, it might keep track of the offset on the decompressed file. OTel filelog receiver, as far as I understood, always start at the beginning or at the end of the file.

| Feature / Vendor       | Vector                                      | OTel filelog receiver                                                                 | Splunk                                                             | FluentBit | DataDog                                         | Fluentd | syslog-ng |
|:-----------------------|:--------------------------------------------|:--------------------------------------------------------------------------------------|:-------------------------------------------------------------------|:----------|:------------------------------------------------|:--------|:----------|
| gzip support           | Yes                                         | Yes                                                                                   | Yes                                                                | No        | No, but Cribl does                              | No      | No        |
| detection              | automatic                                   | Set on config                                                                         | Seems to be based on filename/extension                            | No        | No                                              | No      | No        |
| decompression / stream | "decompress them for reading"               | Stream / standard Go gzip reader                                                      | decompress archive files, prior to processing                      | No        | No / stream                                     | No      | No        |
| Configuration          | auto-detect                                 | set `compression: gzip`                                                               | none / file name                                                   | No        | No / `Content‑Encoding: gzip` connection header | No      | No        |
| Keep state             | Reads once, never again, no real checkpoint | From what I checked in the code: no. `storage` does not say if works with gzip or not | re-ingest if data added. May track offset of the decompressed file | No        | No                                              | No      | No        |


# benchmark

This test case involves reading the first half of a file, then reopening it, seeking to the midpoint, and completing the read operation. In other words, the file is opened a second time, and reading continues from where it left off.

The test utilizes `filestream` mechanisms to read the file line by line, providing a scenario closer to a real-world use case.

The `gzipSeekerReader` is capable of seeking within a gzip file by reading decompressed data up to the required offset. The test employs two `gzipSeekerReader` instances: one to read the initial half, and another to read the remaining portion. The test verifies that the expected number of lines are read to ensure the data is fully read. Relying solely on reading to the end of the file would succeed even if some lines in the middle were missing.

The tests use 2 different approaches to generate log lines:
 - `static-line` generates `line LINE_NUM - line`
 - `random-line` generates `line LINE_NUM - RANDOM TEXT`
It's done to avoid falling into the compression/decompression algorithm best case.

The test/benchmark are on `filebeat/input/filestream/gzip_test.go`.

tl;dr:
```
cd filebeat/input/filestream/
go test -bench=^BenchmarkGzip$ -run=^$ -benchmem
go test -v -run=^TestTimeDifferenceGzipPlain$ .
```
The `-run=^$` is to prevent any test being run

To reproduce the results edit `BenchmarkGzip` to run the benchmark for either
plain text or GZIP. Comment in or out lines 75/76 to configure it.
```go
							// readPlainFile(b, plainFile, half, leftOver, publisher)
							readGZFile(b, gzFile, half, leftOver, publisher)
```
to choose if the benchmark will run for plain file or GZIP. Then run the benchmark
```shell
go test -bench=^BenchmarkGzip$ -run=^$ -benchmem > plain.out
go test -bench=^BenchmarkGzip$ -run=^$ -benchmem > gzip.out
```

To reproduce the results comparing the time difference between the two run the
`TestTimeDifferenceGzipPlain` test:
```shell
go test -v -run=^TestTimeDifferenceGzipPlain$ .
```


## Plain vs GZIP

```text
❯ benchstat plain.mac.out gzip.mac.out
goos: darwin
goarch: arm64
pkg: github.com/elastic/beats/v7/filebeat/input/filestream
cpu: Apple M1 Max
                           │ plain.mac.out │             gzip.mac.out              │
                           │    sec/op     │    sec/op     vs base                 │
Gzip/static-line/1.0_MB-10     12.27m ± 1%    14.78m ± 0%   +20.54% (p=0.000 n=10)
Gzip/static-line/5.2_MB-10     60.05m ± 1%    72.95m ± 0%   +21.48% (p=0.000 n=10)
Gzip/static-line/26_MB-10      300.0m ± 0%    362.5m ± 0%   +20.82% (p=0.000 n=10)
Gzip/static-line/52_MB-10      598.0m ± 0%    723.1m ± 0%   +20.92% (p=0.000 n=10)
Gzip/static-line/105_MB-10      1.199 ± 0%     1.450 ± 2%   +20.90% (p=0.000 n=10)
Gzip/static-line/262_MB-10      3.010 ± 0%     3.616 ± 1%   +20.15% (p=0.000 n=10)
Gzip/static-line/524_MB-10      6.162 ± 0%     7.236 ± 0%   +17.42% (p=0.000 n=10)
Gzip/static-line/1.0_GB-10      12.05 ± 0%     14.53 ± 0%   +20.53% (p=0.000 n=10)
Gzip/random-line/1.0_MB-10     4.890m ± 0%   13.017m ± 0%  +166.18% (p=0.000 n=10)
Gzip/random-line/5.2_MB-10     23.86m ± 0%    63.93m ± 0%  +167.94% (p=0.000 n=10)
Gzip/random-line/26_MB-10      119.7m ± 0%    321.3m ± 0%  +168.47% (p=0.000 n=10)
Gzip/random-line/52_MB-10      240.5m ± 0%    643.9m ± 0%  +167.69% (p=0.000 n=10)
Gzip/random-line/105_MB-10     481.5m ± 0%   1287.3m ± 0%  +167.38% (p=0.000 n=10)
Gzip/random-line/262_MB-10      1.204 ± 0%     3.220 ± 0%  +167.50% (p=0.000 n=10)
Gzip/random-line/524_MB-10      2.411 ± 0%     6.423 ± 0%  +166.46% (p=0.000 n=10)
Gzip/random-line/1.0_GB-10      4.828 ± 0%    13.047 ± 2%  +170.21% (p=0.000 n=10)
geomean                        415.9m         746.6m        +79.49%

                           │ plain.mac.out │             gzip.mac.out             │
                           │     B/op      │     B/op       vs base               │
Gzip/static-line/1.0_MB-10    6.634Mi ± 0%    6.717Mi ± 0%  +1.25% (p=0.000 n=10)
Gzip/static-line/5.2_MB-10    33.57Mi ± 0%    33.65Mi ± 0%  +0.24% (p=0.000 n=10)
Gzip/static-line/26_MB-10     169.5Mi ± 0%    169.6Mi ± 0%  +0.05% (p=0.000 n=10)
Gzip/static-line/52_MB-10     339.2Mi ± 0%    340.6Mi ± 0%  +0.42% (p=0.000 n=10)
Gzip/static-line/105_MB-10    679.8Mi ± 0%    682.6Mi ± 0%  +0.41% (p=0.000 n=10)
Gzip/static-line/262_MB-10    1.676Gi ± 0%    1.677Gi ± 0%  +0.03% (p=0.000 n=10)
Gzip/static-line/524_MB-10    3.364Gi ± 0%    3.364Gi ± 0%  +0.00% (p=0.000 n=10)
Gzip/static-line/1.0_GB-10    6.738Gi ± 0%    6.738Gi ± 0%  +0.00% (p=0.000 n=10)
Gzip/random-line/1.0_MB-10    3.963Mi ± 0%    4.065Mi ± 0%  +2.58% (p=0.000 n=10)
Gzip/random-line/5.2_MB-10    19.93Mi ± 0%    20.05Mi ± 0%  +0.61% (p=0.000 n=10)
Gzip/random-line/26_MB-10     100.3Mi ± 0%    100.3Mi ± 0%  +0.08% (p=0.000 n=10)
Gzip/random-line/52_MB-10     201.0Mi ± 0%    201.1Mi ± 0%  +0.04% (p=0.000 n=10)
Gzip/random-line/105_MB-10    403.1Mi ± 0%    403.2Mi ± 0%  +0.02% (p=0.000 n=10)
Gzip/random-line/262_MB-10   1009.4Mi ± 0%   1009.5Mi ± 0%  +0.01% (p=0.000 n=10)
Gzip/random-line/524_MB-10    1.978Gi ± 0%    1.978Gi ± 0%  +0.00% (p=0.000 n=10)
Gzip/random-line/1.0_GB-10    3.971Gi ± 0%    3.971Gi ± 0%  +0.00% (p=0.000 n=10)
geomean                       285.0Mi         286.0Mi       +0.36%

                           │ plain.mac.out │            gzip.mac.out            │
                           │   allocs/op   │  allocs/op   vs base               │
Gzip/static-line/1.0_MB-10     150.2k ± 0%   150.2k ± 0%  +0.02% (p=0.000 n=10)
Gzip/static-line/5.2_MB-10     750.8k ± 0%   750.8k ± 0%  +0.01% (p=0.000 n=10)
Gzip/static-line/26_MB-10      3.754M ± 0%   3.754M ± 0%  +0.00% (p=0.000 n=10)
Gzip/static-line/52_MB-10      7.508M ± 0%   7.508M ± 0%  +0.00% (p=0.000 n=10)
Gzip/static-line/105_MB-10     15.02M ± 0%   15.02M ± 0%  +0.00% (p=0.000 n=10)
Gzip/static-line/262_MB-10     37.54M ± 0%   37.54M ± 0%  +0.00% (p=0.000 n=10)
Gzip/static-line/524_MB-10     75.09M ± 0%   75.09M ± 0%  +0.00% (p=0.000 n=10)
Gzip/static-line/1.0_GB-10     150.2M ± 0%   150.2M ± 0%  +0.00% (p=0.000 n=10)
Gzip/random-line/1.0_MB-10     58.59k ± 0%   58.62k ± 0%  +0.06% (p=0.000 n=10)
Gzip/random-line/5.2_MB-10     292.8k ± 0%   292.8k ± 0%  +0.01% (p=0.000 n=10)
Gzip/random-line/26_MB-10      1.464M ± 0%   1.464M ± 0%  +0.00% (p=0.000 n=10)
Gzip/random-line/52_MB-10      2.928M ± 0%   2.928M ± 0%  +0.00% (p=0.000 n=10)
Gzip/random-line/105_MB-10     5.855M ± 0%   5.855M ± 0%  +0.00% (p=0.000 n=10)
Gzip/random-line/262_MB-10     14.64M ± 0%   14.64M ± 0%  +0.00% (p=0.000 n=10)
Gzip/random-line/524_MB-10     29.28M ± 0%   29.28M ± 0%  +0.00% (p=0.001 n=10)
Gzip/random-line/1.0_GB-10     58.56M ± 0%   58.56M ± 0%  +0.00% (p=0.002 n=10)
geomean                        5.113M        5.113M       +0.01%
```

### Plain text
```
goos: darwin
goarch: arm64
pkg: github.com/elastic/beats/v7/filebeat/input/filestream
cpu: Apple M1 Max
BenchmarkGzip/static-line/1.0_MB-10  	      94	  12154500 ns/op	 6956338 B/op	  150173 allocs/op
BenchmarkGzip/static-line/1.0_MB-10  	      97	  12340168 ns/op	 6956352 B/op	  150173 allocs/op
BenchmarkGzip/static-line/1.0_MB-10  	      97	  12193341 ns/op	 6956288 B/op	  150173 allocs/op
BenchmarkGzip/static-line/1.0_MB-10  	      96	  12255799 ns/op	 6956299 B/op	  150173 allocs/op
BenchmarkGzip/static-line/1.0_MB-10  	      97	  12358144 ns/op	 6956277 B/op	  150173 allocs/op
BenchmarkGzip/static-line/1.0_MB-10  	      97	  12289761 ns/op	 6956284 B/op	  150173 allocs/op
BenchmarkGzip/static-line/1.0_MB-10  	      96	  12269043 ns/op	 6956303 B/op	  150173 allocs/op
BenchmarkGzip/static-line/1.0_MB-10  	      98	  12261883 ns/op	 6956287 B/op	  150173 allocs/op
BenchmarkGzip/static-line/1.0_MB-10  	      99	  12224223 ns/op	 6956282 B/op	  150173 allocs/op
BenchmarkGzip/static-line/1.0_MB-10  	      98	  12324082 ns/op	 6956296 B/op	  150173 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      19	  60470129 ns/op	35199584 B/op	  750764 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      19	  60063572 ns/op	35199493 B/op	  750763 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      19	  60448303 ns/op	35199594 B/op	  750764 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      19	  59906007 ns/op	35199509 B/op	  750764 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      19	  60010564 ns/op	35199488 B/op	  750763 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      19	  59944298 ns/op	35199567 B/op	  750764 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      19	  59881493 ns/op	35199494 B/op	  750763 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      19	  60033079 ns/op	35199472 B/op	  750763 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      19	  60200474 ns/op	35199534 B/op	  750764 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      19	  60251331 ns/op	35199504 B/op	  750763 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       4	 299859667 ns/op	177770884 B/op	 3753957 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       4	 300392948 ns/op	177771208 B/op	 3753960 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       4	 299847021 ns/op	177771376 B/op	 3753961 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       4	 300436916 ns/op	177771292 B/op	 3753961 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       4	 300325958 ns/op	177771328 B/op	 3753961 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       4	 300322708 ns/op	177770860 B/op	 3753956 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       4	 300182479 ns/op	177771616 B/op	 3753964 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       4	 299683823 ns/op	177771148 B/op	 3753959 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       4	 299325312 ns/op	177771400 B/op	 3753962 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       4	 299634854 ns/op	177771328 B/op	 3753961 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 597444792 ns/op	355659960 B/op	 7508148 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 598162667 ns/op	355660440 B/op	 7508153 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 598249750 ns/op	355659048 B/op	 7508139 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 598603062 ns/op	355660104 B/op	 7508150 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 598137666 ns/op	355660152 B/op	 7508150 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 598879458 ns/op	355660104 B/op	 7508150 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 597810812 ns/op	355659960 B/op	 7508148 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 597607500 ns/op	355660152 B/op	 7508150 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 597342562 ns/op	355660248 B/op	 7508151 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 597611084 ns/op	355660392 B/op	 7508153 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1199693084 ns/op	712842264 B/op	15016547 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1201442500 ns/op	712839096 B/op	15016514 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1198444750 ns/op	712840632 B/op	15016530 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1200589458 ns/op	712840344 B/op	15016527 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1197789250 ns/op	712840248 B/op	15016526 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1198706292 ns/op	712841208 B/op	15016536 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1198450208 ns/op	712839672 B/op	15016520 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1200287292 ns/op	712841496 B/op	15016539 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1198466542 ns/op	712840728 B/op	15016531 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1199425083 ns/op	712841784 B/op	15016542 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3011090750 ns/op	1799647792 B/op	37543781 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3010731125 ns/op	1799652112 B/op	37543826 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3004944042 ns/op	1799648848 B/op	37543792 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3005438625 ns/op	1799650192 B/op	37543806 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3012704791 ns/op	1799650384 B/op	37543808 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3008927292 ns/op	1799648320 B/op	37543788 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3010444042 ns/op	1799647984 B/op	37543786 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3012376542 ns/op	1799646064 B/op	37543766 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3007852875 ns/op	1799645392 B/op	37543759 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3008176500 ns/op	1799650192 B/op	37543806 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	6133979500 ns/op	3611780232 B/op	75090387 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	6165572833 ns/op	3611783400 B/op	75090420 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	6164568000 ns/op	3611775672 B/op	75090338 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	6160973250 ns/op	3611779080 B/op	75090375 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	6173552333 ns/op	3611774616 B/op	75090330 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	6163018875 ns/op	3611774760 B/op	75090330 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	6168454917 ns/op	3611779176 B/op	75090376 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	6155820250 ns/op	3611774424 B/op	75090328 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	6156142541 ns/op	3611775048 B/op	75090333 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	6156358708 ns/op	3611776200 B/op	75090345 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	12074229792 ns/op	7235062176 B/op	150183265 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	12176074417 ns/op	7235068608 B/op	150183341 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	12056455375 ns/op	7235055024 B/op	150183192 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	12045805833 ns/op	7235056368 B/op	150183206 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	12057154250 ns/op	7235055936 B/op	150183203 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	12050246792 ns/op	7235061312 B/op	150183256 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	12052299417 ns/op	7235058192 B/op	150183225 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	12043996208 ns/op	7235060976 B/op	150183251 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	12047884125 ns/op	7235054688 B/op	150183190 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	12053419917 ns/op	7235058336 B/op	150183228 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	     243	   4890378 ns/op	 4155180 B/op	   58585 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	     244	   4887025 ns/op	 4155183 B/op	   58585 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	     243	   4894869 ns/op	 4155180 B/op	   58585 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	     243	   4890534 ns/op	 4155182 B/op	   58585 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	     243	   4931099 ns/op	 4155184 B/op	   58585 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	     244	   4893987 ns/op	 4155185 B/op	   58585 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	     244	   4889625 ns/op	 4155188 B/op	   58585 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	     243	   4890390 ns/op	 4155183 B/op	   58585 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	     244	   4895556 ns/op	 4155186 B/op	   58585 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	     244	   4880850 ns/op	 4155170 B/op	   58585 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      50	  23830786 ns/op	20901213 B/op	  292763 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      50	  23847700 ns/op	20901176 B/op	  292762 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      50	  23873462 ns/op	20901240 B/op	  292763 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      50	  23821638 ns/op	20901238 B/op	  292763 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      50	  23886369 ns/op	20901223 B/op	  292763 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      49	  23858837 ns/op	20901173 B/op	  292762 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      50	  23862815 ns/op	20901241 B/op	  292763 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      50	  23898386 ns/op	20901247 B/op	  292763 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      50	  23826028 ns/op	20901189 B/op	  292762 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      50	  23871190 ns/op	20901245 B/op	  292763 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       9	 119669917 ns/op	105121658 B/op	 1463743 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       9	 119827528 ns/op	105121621 B/op	 1463743 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       9	 119654009 ns/op	105121669 B/op	 1463743 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       9	 119584139 ns/op	105121381 B/op	 1463740 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       9	 119991616 ns/op	105121669 B/op	 1463744 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       9	 119618954 ns/op	105121434 B/op	 1463741 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       9	 119686468 ns/op	105121637 B/op	 1463743 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       9	 119657273 ns/op	105121562 B/op	 1463742 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       9	 119736736 ns/op	105121797 B/op	 1463745 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       9	 119876227 ns/op	105121178 B/op	 1463739 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       5	 240480683 ns/op	210799686 B/op	 2927579 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       5	 240751333 ns/op	210799897 B/op	 2927582 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       5	 240294792 ns/op	210799984 B/op	 2927582 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       5	 240431550 ns/op	210799820 B/op	 2927581 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       5	 240450475 ns/op	210799763 B/op	 2927580 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       5	 240873517 ns/op	210799532 B/op	 2927578 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       5	 240324833 ns/op	210799772 B/op	 2927580 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       5	 240572058 ns/op	210799542 B/op	 2927578 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       5	 240787917 ns/op	210800041 B/op	 2927583 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       5	 240889783 ns/op	210800051 B/op	 2927583 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       3	 481690917 ns/op	422709312 B/op	 5855400 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       3	 481953875 ns/op	422708448 B/op	 5855391 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       3	 481535250 ns/op	422707888 B/op	 5855385 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       3	 480904361 ns/op	422708208 B/op	 5855388 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       3	 481844430 ns/op	422707808 B/op	 5855384 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       3	 480967958 ns/op	422708576 B/op	 5855392 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       3	 480682417 ns/op	422707760 B/op	 5855383 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       3	 481899542 ns/op	422707952 B/op	 5855386 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       3	 481376472 ns/op	422707152 B/op	 5855378 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       3	 480717903 ns/op	422708464 B/op	 5855391 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	1203794917 ns/op	1058425808 B/op	14638821 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	1202207625 ns/op	1058425424 B/op	14638817 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	1203609167 ns/op	1058424560 B/op	14638808 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	1206076833 ns/op	1058426864 B/op	14638835 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	1204044042 ns/op	1058424512 B/op	14638809 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	1204613500 ns/op	1058424272 B/op	14638808 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	1201633208 ns/op	1058424224 B/op	14638806 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	1203704875 ns/op	1058424368 B/op	14638806 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	1203467125 ns/op	1058424464 B/op	14638807 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	1203339584 ns/op	1058425424 B/op	14638817 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	2410131083 ns/op	2124223952 B/op	29278973 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	2411492958 ns/op	2124225872 B/op	29278996 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	2408864750 ns/op	2124224576 B/op	29278984 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	2407980625 ns/op	2124222224 B/op	29278958 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	2413095000 ns/op	2124225056 B/op	29278986 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	2413195416 ns/op	2124224336 B/op	29278980 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	2407794000 ns/op	2124221024 B/op	29278944 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	2411699375 ns/op	2124223568 B/op	29278972 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	2409607042 ns/op	2124225728 B/op	29278993 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	2411187000 ns/op	2124223952 B/op	29278976 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	4841395250 ns/op	4263303120 B/op	58560635 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	4824481500 ns/op	4263298608 B/op	58560588 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	4833240917 ns/op	4263296112 B/op	58560565 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	4829002125 ns/op	4263301152 B/op	58560616 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	4825928292 ns/op	4263296688 B/op	58560571 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	4827327417 ns/op	4263302688 B/op	58560632 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	4824214375 ns/op	4263303216 B/op	58560633 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	4832651125 ns/op	4263297072 B/op	58560575 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	4827801250 ns/op	4263307968 B/op	58560687 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	4831864958 ns/op	4263298272 B/op	58560583 allocs/op
PASS
ok  	github.com/elastic/beats/v7/filebeat/input/filestream	1032.288s
```

### GZIP
```
goos: darwin
goarch: arm64
pkg: github.com/elastic/beats/v7/filebeat/input/filestream
cpu: Apple M1 Max
BenchmarkGzip/static-line/1.0_MB-10  	      80	  14729888 ns/op	 7043306 B/op	  150204 allocs/op
BenchmarkGzip/static-line/1.0_MB-10  	      74	  14936529 ns/op	 7043329 B/op	  150204 allocs/op
BenchmarkGzip/static-line/1.0_MB-10  	      80	  14800536 ns/op	 7043171 B/op	  150204 allocs/op
BenchmarkGzip/static-line/1.0_MB-10  	      80	  14774578 ns/op	 7043186 B/op	  150204 allocs/op
BenchmarkGzip/static-line/1.0_MB-10  	      81	  14818237 ns/op	 7043170 B/op	  150204 allocs/op
BenchmarkGzip/static-line/1.0_MB-10  	      80	  14781417 ns/op	 7043171 B/op	  150204 allocs/op
BenchmarkGzip/static-line/1.0_MB-10  	      81	  14745377 ns/op	 7043166 B/op	  150204 allocs/op
BenchmarkGzip/static-line/1.0_MB-10  	      79	  14776437 ns/op	 7043164 B/op	  150204 allocs/op
BenchmarkGzip/static-line/1.0_MB-10  	      81	  14796556 ns/op	 7043188 B/op	  150204 allocs/op
BenchmarkGzip/static-line/1.0_MB-10  	      81	  14788148 ns/op	 7043183 B/op	  150204 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      15	  72814861 ns/op	35285664 B/op	  750813 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      15	  72997997 ns/op	35285697 B/op	  750814 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      15	  72939042 ns/op	35285689 B/op	  750813 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      15	  73073828 ns/op	35285665 B/op	  750813 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      15	  72817167 ns/op	35285740 B/op	  750814 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      15	  73100011 ns/op	35285712 B/op	  750814 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      15	  72946386 ns/op	35285753 B/op	  750814 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      15	  72945158 ns/op	35285741 B/op	  750814 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      15	  73152083 ns/op	35285677 B/op	  750813 allocs/op
BenchmarkGzip/static-line/5.2_MB-10  	      15	  72822428 ns/op	35285698 B/op	  750814 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       3	 363193319 ns/op	177860165 B/op	 3754059 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       3	 362947569 ns/op	177859850 B/op	 3754055 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       3	 362581889 ns/op	177859658 B/op	 3754053 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       3	 362025542 ns/op	177859813 B/op	 3754055 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       3	 363021000 ns/op	177859706 B/op	 3754054 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       3	 362199417 ns/op	177860266 B/op	 3754060 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       3	 362660542 ns/op	177859690 B/op	 3754054 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       3	 361424611 ns/op	177859626 B/op	 3754053 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       3	 362390764 ns/op	177859728 B/op	 3754054 allocs/op
BenchmarkGzip/static-line/26_MB-10   	       3	 361467347 ns/op	177860016 B/op	 3754057 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 724741646 ns/op	357152760 B/op	 7508250 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 723723520 ns/op	357153432 B/op	 7508257 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 723232062 ns/op	357153000 B/op	 7508252 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 722075396 ns/op	357152976 B/op	 7508251 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 722113375 ns/op	357153096 B/op	 7508253 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 723437625 ns/op	357151760 B/op	 7508239 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 722668812 ns/op	357152096 B/op	 7508243 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 722905000 ns/op	357153144 B/op	 7508254 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 723689708 ns/op	357153056 B/op	 7508253 allocs/op
BenchmarkGzip/static-line/52_MB-10   	       2	 722283938 ns/op	357153456 B/op	 7508256 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1445928459 ns/op	715736232 B/op	15016654 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1446721792 ns/op	715736584 B/op	15016656 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1447822292 ns/op	715737784 B/op	15016670 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1451548083 ns/op	715738456 B/op	15016677 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1478452083 ns/op	715740280 B/op	15016696 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1469530959 ns/op	715737304 B/op	15016665 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1472564000 ns/op	715738552 B/op	15016678 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1456403958 ns/op	715738696 B/op	15016678 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1445475750 ns/op	715736632 B/op	15016658 allocs/op
BenchmarkGzip/static-line/105_MB-10  	       1	1445874250 ns/op	715737256 B/op	15016663 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3616675834 ns/op	1800218864 B/op	37543997 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3618013167 ns/op	1800217424 B/op	37543982 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3613785042 ns/op	1800217616 B/op	37543984 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3613869208 ns/op	1800217968 B/op	37543986 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3608353625 ns/op	1800218672 B/op	37543995 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3615753958 ns/op	1800217856 B/op	37543985 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3616759750 ns/op	1800216944 B/op	37543977 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3690132333 ns/op	1800221216 B/op	37544026 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3646990792 ns/op	1800218880 B/op	37543997 allocs/op
BenchmarkGzip/static-line/262_MB-10  	       1	3614083208 ns/op	1800213632 B/op	37543941 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	7230062875 ns/op	3611865896 B/op	75090536 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	7253560417 ns/op	3611864232 B/op	75090520 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	7238066500 ns/op	3611866200 B/op	75090542 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	7246401125 ns/op	3611865240 B/op	75090529 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	7229662750 ns/op	3611857048 B/op	75090445 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	7226772833 ns/op	3611863000 B/op	75090507 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	7233631167 ns/op	3611864616 B/op	75090524 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	7227619292 ns/op	3611865288 B/op	75090531 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	7245732875 ns/op	3611862840 B/op	75090507 allocs/op
BenchmarkGzip/static-line/524_MB-10  	       1	7237774083 ns/op	3611856792 B/op	75090444 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	14493834333 ns/op	7235150784 B/op	150183508 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	14487530084 ns/op	7235157360 B/op	150183581 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	14470533500 ns/op	7235157856 B/op	150183586 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	14613722000 ns/op	7235158224 B/op	150183596 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	14591394500 ns/op	7235160640 B/op	150183621 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	14464514459 ns/op	7235154528 B/op	150183556 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	14492094000 ns/op	7235152992 B/op	150183537 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	14576413083 ns/op	7235158496 B/op	150183599 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	14559584292 ns/op	7235166128 B/op	150183674 allocs/op
BenchmarkGzip/static-line/1.0_GB-10  	       1	14571519584 ns/op	7235162448 B/op	150183634 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	      91	  13002107 ns/op	 4262237 B/op	   58619 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	      91	  13022501 ns/op	 4262228 B/op	   58619 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	      92	  13009575 ns/op	 4262236 B/op	   58619 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	      91	  13024327 ns/op	 4262239 B/op	   58619 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	      92	  13017265 ns/op	 4262224 B/op	   58619 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	      92	  13011312 ns/op	 4262222 B/op	   58619 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	      91	  13017429 ns/op	 4262234 B/op	   58619 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	      92	  13006923 ns/op	 4262231 B/op	   58619 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	      92	  13081588 ns/op	 4262228 B/op	   58619 allocs/op
BenchmarkGzip/random-line/1.0_MB-10  	      91	  13121174 ns/op	 4262233 B/op	   58619 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      18	  63914025 ns/op	21028377 B/op	  292796 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      18	  64010995 ns/op	21028395 B/op	  292797 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      18	  64105595 ns/op	21028409 B/op	  292797 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      18	  64176164 ns/op	21028426 B/op	  292797 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      18	  63951255 ns/op	21028372 B/op	  292796 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      18	  63913130 ns/op	21028443 B/op	  292797 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      18	  63963049 ns/op	21028371 B/op	  292796 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      18	  63900965 ns/op	21028446 B/op	  292797 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      18	  63877403 ns/op	21028425 B/op	  292797 allocs/op
BenchmarkGzip/random-line/5.2_MB-10  	      18	  63890958 ns/op	21028434 B/op	  292797 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       4	 326835688 ns/op	105207144 B/op	 1463776 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       4	 322185146 ns/op	105207464 B/op	 1463779 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       4	 321168927 ns/op	105207096 B/op	 1463775 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       4	 321419302 ns/op	105207648 B/op	 1463781 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       4	 321431812 ns/op	105207100 B/op	 1463775 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       4	 321130062 ns/op	105207260 B/op	 1463777 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       4	 321066990 ns/op	105207504 B/op	 1463779 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       4	 321190062 ns/op	105207384 B/op	 1463778 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       4	 321522927 ns/op	105207196 B/op	 1463776 allocs/op
BenchmarkGzip/random-line/26_MB-10   	       4	 321035604 ns/op	105207076 B/op	 1463775 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       2	 643574958 ns/op	210887176 B/op	 2927628 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       2	 643328938 ns/op	210887440 B/op	 2927630 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       2	 643431771 ns/op	210886696 B/op	 2927623 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       2	 643445896 ns/op	210887176 B/op	 2927628 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       2	 645074812 ns/op	210886936 B/op	 2927625 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       2	 645847500 ns/op	210887320 B/op	 2927629 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       2	 644080208 ns/op	210887248 B/op	 2927628 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       2	 647828396 ns/op	210887224 B/op	 2927628 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       2	 644467146 ns/op	210887032 B/op	 2927626 allocs/op
BenchmarkGzip/random-line/52_MB-10   	       2	 643647396 ns/op	210887176 B/op	 2927628 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       1	1286879500 ns/op	422793248 B/op	 5855415 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       1	1287103792 ns/op	422794288 B/op	 5855426 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       1	1287382584 ns/op	422794000 B/op	 5855423 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       1	1288113292 ns/op	422794672 B/op	 5855430 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       1	1287247541 ns/op	422794288 B/op	 5855426 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       1	1285914917 ns/op	422794576 B/op	 5855429 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       1	1287366667 ns/op	422794960 B/op	 5855433 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       1	1288303042 ns/op	422794768 B/op	 5855431 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       1	1286533458 ns/op	422795248 B/op	 5855436 allocs/op
BenchmarkGzip/random-line/105_MB-10  	       1	1288886917 ns/op	422795248 B/op	 5855436 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	3218411000 ns/op	1058510384 B/op	14638853 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	3219338042 ns/op	1058510864 B/op	14638858 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	3221438958 ns/op	1058510576 B/op	14638855 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	3218982875 ns/op	1058509136 B/op	14638840 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	3219704375 ns/op	1058510288 B/op	14638852 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	3220671333 ns/op	1058509616 B/op	14638845 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	3220505458 ns/op	1058510144 B/op	14638849 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	3217060833 ns/op	1058509904 B/op	14638848 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	3221287125 ns/op	1058509232 B/op	14638841 allocs/op
BenchmarkGzip/random-line/262_MB-10  	       1	3219768750 ns/op	1058510624 B/op	14638854 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	6427231458 ns/op	2124308080 B/op	29279002 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	6423423167 ns/op	2124309280 B/op	29279016 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	6423273375 ns/op	2124309120 B/op	29279013 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	6425959625 ns/op	2124308832 B/op	29279010 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	6420208000 ns/op	2124309808 B/op	29279020 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	6427532084 ns/op	2124307968 B/op	29279001 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	6423558792 ns/op	2124308064 B/op	29279002 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	6418181750 ns/op	2124307008 B/op	29278991 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	6420053041 ns/op	2124305232 B/op	29278974 allocs/op
BenchmarkGzip/random-line/524_MB-10  	       1	6435926667 ns/op	2124307104 B/op	29278992 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	12875916208 ns/op	4263384112 B/op	58560649 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	12830571084 ns/op	4263387760 B/op	58560690 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	12832356250 ns/op	4263376912 B/op	58560577 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	12830545833 ns/op	4263386672 B/op	58560677 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	13073027250 ns/op	4263387200 B/op	58560684 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	13103952209 ns/op	4263389104 B/op	58560704 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	13120141583 ns/op	4263391232 B/op	58560726 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	13125795250 ns/op	4263387520 B/op	58560689 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	13117050292 ns/op	4263392416 B/op	58560740 allocs/op
BenchmarkGzip/random-line/1.0_GB-10  	       1	13020225958 ns/op	4263388624 B/op	58560699 allocs/op
PASS
ok  	github.com/elastic/beats/v7/filebeat/input/filestream	1212.282s
```

### The time difference between both options
```
❯ go test -v -run=^TestTimeDifferenceGzipPlain$ -timeout=0 .
=== RUN   TestTimeDifferenceGzipPlain
=== RUN   TestTimeDifferenceGzipPlain/static-line
    gzip_test.go:357: plain file to be gzipped has 1.3 MB
=== RUN   TestTimeDifferenceGzipPlain/static-line/1.0_MB-plain-file
    gzip_test.go:116: readPlainFile took 18.735234ms for 74899 lines
=== RUN   TestTimeDifferenceGzipPlain/static-line/1.0_MB-gzip-file
    gzip_test.go:124: readGZFile took 26.942668ms for 74899 lines
=== NAME  TestTimeDifferenceGzipPlain/static-line
    gzip_test.go:127:
        static-line: for 1.0 MB, gzip read took 8.207434ms more than pain read

    gzip_test.go:357: plain file to be gzipped has 7.0 MB
=== RUN   TestTimeDifferenceGzipPlain/static-line/5.2_MB-plain-file
    gzip_test.go:116: readPlainFile took 104.661779ms for 374492 lines
=== RUN   TestTimeDifferenceGzipPlain/static-line/5.2_MB-gzip-file
    gzip_test.go:124: readGZFile took 126.344924ms for 374492 lines
=== NAME  TestTimeDifferenceGzipPlain/static-line
    gzip_test.go:127:
        static-line: for 5.2 MB, gzip read took 21.683145ms more than pain read

    gzip_test.go:357: plain file to be gzipped has 36 MB
=== RUN   TestTimeDifferenceGzipPlain/static-line/26_MB-plain-file
    gzip_test.go:116: readPlainFile took 483.838109ms for 1872458 lines
=== RUN   TestTimeDifferenceGzipPlain/static-line/26_MB-gzip-file
    gzip_test.go:124: readGZFile took 553.314773ms for 1872458 lines
=== NAME  TestTimeDifferenceGzipPlain/static-line
    gzip_test.go:127:
        static-line: for 26 MB, gzip read took 69.476664ms more than pain read

    gzip_test.go:357: plain file to be gzipped has 74 MB
=== RUN   TestTimeDifferenceGzipPlain/static-line/52_MB-plain-file
    gzip_test.go:116: readPlainFile took 1.017979376s for 3744915 lines
=== RUN   TestTimeDifferenceGzipPlain/static-line/52_MB-gzip-file
    gzip_test.go:124: readGZFile took 1.168560421s for 3744915 lines
=== NAME  TestTimeDifferenceGzipPlain/static-line
    gzip_test.go:127:
        static-line: for 52 MB, gzip read took 150.581045ms more than pain read

    gzip_test.go:357: plain file to be gzipped has 149 MB
=== RUN   TestTimeDifferenceGzipPlain/static-line/105_MB-plain-file
    gzip_test.go:116: readPlainFile took 2.265440104s for 7489829 lines
=== RUN   TestTimeDifferenceGzipPlain/static-line/105_MB-gzip-file
    gzip_test.go:124: readGZFile took 2.532667971s for 7489829 lines
=== NAME  TestTimeDifferenceGzipPlain/static-line
    gzip_test.go:127:
        static-line: for 105 MB, gzip read took 267.227867ms more than pain read

    gzip_test.go:357: plain file to be gzipped has 382 MB
=== RUN   TestTimeDifferenceGzipPlain/static-line/262_MB-plain-file
    gzip_test.go:116: readPlainFile took 5.33261602s for 18724572 lines
=== RUN   TestTimeDifferenceGzipPlain/static-line/262_MB-gzip-file
    gzip_test.go:124: readGZFile took 6.160383896s for 18724572 lines
=== NAME  TestTimeDifferenceGzipPlain/static-line
    gzip_test.go:127:
        static-line: for 262 MB, gzip read took 827.767876ms more than pain read

    gzip_test.go:357: plain file to be gzipped has 775 MB
=== RUN   TestTimeDifferenceGzipPlain/static-line/524_MB-plain-file
    gzip_test.go:116: readPlainFile took 9.913055724s for 37449143 lines
=== RUN   TestTimeDifferenceGzipPlain/static-line/524_MB-gzip-file
    gzip_test.go:124: readGZFile took 11.962886825s for 37449143 lines
=== NAME  TestTimeDifferenceGzipPlain/static-line
    gzip_test.go:127:
        static-line: for 524 MB, gzip read took 2.049831101s more than pain read

    gzip_test.go:357: plain file to be gzipped has 1.6 GB
=== RUN   TestTimeDifferenceGzipPlain/static-line/1.0_GB-plain-file
    gzip_test.go:116: readPlainFile took 19.318297604s for 74898286 lines
=== RUN   TestTimeDifferenceGzipPlain/static-line/1.0_GB-gzip-file
    gzip_test.go:124: readGZFile took 23.36719171s for 74898286 lines
=== NAME  TestTimeDifferenceGzipPlain/static-line
    gzip_test.go:127:
        static-line: for 1.0 GB, gzip read took 4.048894106s more than pain read

=== RUN   TestTimeDifferenceGzipPlain/random-line
    gzip_test.go:357: plain file to be gzipped has 1.2 MB
=== RUN   TestTimeDifferenceGzipPlain/random-line/1.0_MB-plain-file
    gzip_test.go:116: readPlainFile took 9.32454ms for 29128 lines
=== RUN   TestTimeDifferenceGzipPlain/random-line/1.0_MB-gzip-file
    gzip_test.go:124: readGZFile took 18.964286ms for 29128 lines
=== NAME  TestTimeDifferenceGzipPlain/random-line
    gzip_test.go:127:
        random-line: for 1.0 MB, gzip read took 9.639746ms more than pain read

    gzip_test.go:357: plain file to be gzipped has 5.9 MB
=== RUN   TestTimeDifferenceGzipPlain/random-line/5.2_MB-plain-file
    gzip_test.go:116: readPlainFile took 43.750972ms for 145636 lines
=== RUN   TestTimeDifferenceGzipPlain/random-line/5.2_MB-gzip-file
    gzip_test.go:124: readGZFile took 96.184701ms for 145636 lines
=== NAME  TestTimeDifferenceGzipPlain/random-line
    gzip_test.go:127:
        random-line: for 5.2 MB, gzip read took 52.433729ms more than pain read

    gzip_test.go:357: plain file to be gzipped has 30 MB
=== RUN   TestTimeDifferenceGzipPlain/random-line/26_MB-plain-file
    gzip_test.go:116: readPlainFile took 231.944553ms for 728178 lines
=== RUN   TestTimeDifferenceGzipPlain/random-line/26_MB-gzip-file
    gzip_test.go:124: readGZFile took 489.032069ms for 728178 lines
=== NAME  TestTimeDifferenceGzipPlain/random-line
    gzip_test.go:127:
        random-line: for 26 MB, gzip read took 257.087516ms more than pain read

    gzip_test.go:357: plain file to be gzipped has 60 MB
=== RUN   TestTimeDifferenceGzipPlain/random-line/52_MB-plain-file
    gzip_test.go:116: readPlainFile took 442.882148ms for 1456356 lines
=== RUN   TestTimeDifferenceGzipPlain/random-line/52_MB-gzip-file
    gzip_test.go:124: readGZFile took 997.63121ms for 1456356 lines
=== NAME  TestTimeDifferenceGzipPlain/random-line
    gzip_test.go:127:
        random-line: for 52 MB, gzip read took 554.749062ms more than pain read

    gzip_test.go:357: plain file to be gzipped has 121 MB
=== RUN   TestTimeDifferenceGzipPlain/random-line/105_MB-plain-file
    gzip_test.go:116: readPlainFile took 1.072072924s for 2912712 lines
=== RUN   TestTimeDifferenceGzipPlain/random-line/105_MB-gzip-file
    gzip_test.go:124: readGZFile took 2.210950489s for 2912712 lines
=== NAME  TestTimeDifferenceGzipPlain/random-line
    gzip_test.go:127:
        random-line: for 105 MB, gzip read took 1.138877565s more than pain read

    gzip_test.go:357: plain file to be gzipped has 305 MB
=== RUN   TestTimeDifferenceGzipPlain/random-line/262_MB-plain-file
    gzip_test.go:116: readPlainFile took 2.28257804s for 7281778 lines
=== RUN   TestTimeDifferenceGzipPlain/random-line/262_MB-gzip-file
    gzip_test.go:124: readGZFile took 5.318570361s for 7281778 lines
=== NAME  TestTimeDifferenceGzipPlain/random-line
    gzip_test.go:127:
        random-line: for 262 MB, gzip read took 3.035992321s more than pain read

    gzip_test.go:357: plain file to be gzipped has 615 MB
=== RUN   TestTimeDifferenceGzipPlain/random-line/524_MB-plain-file
    gzip_test.go:116: readPlainFile took 4.178564272s for 14563556 lines
=== RUN   TestTimeDifferenceGzipPlain/random-line/524_MB-gzip-file
    gzip_test.go:124: readGZFile took 9.7902523s for 14563556 lines
=== NAME  TestTimeDifferenceGzipPlain/random-line
    gzip_test.go:127:
        random-line: for 524 MB, gzip read took 5.611688028s more than pain read

    gzip_test.go:357: plain file to be gzipped has 1.2 GB
=== RUN   TestTimeDifferenceGzipPlain/random-line/1.0_GB-plain-file
    gzip_test.go:116: readPlainFile took 10.129862815s for 29127112 lines
=== RUN   TestTimeDifferenceGzipPlain/random-line/1.0_GB-gzip-file
    gzip_test.go:124: readGZFile took 19.555034385s for 29127112 lines
=== NAME  TestTimeDifferenceGzipPlain/random-line
    gzip_test.go:127:
        random-line: for 1.0 GB, gzip read took 9.42517157s more than pain read

--- PASS: TestTimeDifferenceGzipPlain (1250.54s)
    --- PASS: TestTimeDifferenceGzipPlain/static-line (808.73s)
        --- PASS: TestTimeDifferenceGzipPlain/static-line/1.0_MB-plain-file (0.02s)
        --- PASS: TestTimeDifferenceGzipPlain/static-line/1.0_MB-gzip-file (0.03s)
        --- PASS: TestTimeDifferenceGzipPlain/static-line/5.2_MB-plain-file (0.10s)
        --- PASS: TestTimeDifferenceGzipPlain/static-line/5.2_MB-gzip-file (0.13s)
        --- PASS: TestTimeDifferenceGzipPlain/static-line/26_MB-plain-file (0.48s)
        --- PASS: TestTimeDifferenceGzipPlain/static-line/26_MB-gzip-file (0.55s)
        --- PASS: TestTimeDifferenceGzipPlain/static-line/52_MB-plain-file (1.02s)
        --- PASS: TestTimeDifferenceGzipPlain/static-line/52_MB-gzip-file (1.17s)
        --- PASS: TestTimeDifferenceGzipPlain/static-line/105_MB-plain-file (2.27s)
        --- PASS: TestTimeDifferenceGzipPlain/static-line/105_MB-gzip-file (2.53s)
        --- PASS: TestTimeDifferenceGzipPlain/static-line/262_MB-plain-file (5.33s)
        --- PASS: TestTimeDifferenceGzipPlain/static-line/262_MB-gzip-file (6.16s)
        --- PASS: TestTimeDifferenceGzipPlain/static-line/524_MB-plain-file (9.91s)
        --- PASS: TestTimeDifferenceGzipPlain/static-line/524_MB-gzip-file (11.96s)
        --- PASS: TestTimeDifferenceGzipPlain/static-line/1.0_GB-plain-file (19.32s)
        --- PASS: TestTimeDifferenceGzipPlain/static-line/1.0_GB-gzip-file (23.37s)
    --- PASS: TestTimeDifferenceGzipPlain/random-line (441.81s)
        --- PASS: TestTimeDifferenceGzipPlain/random-line/1.0_MB-plain-file (0.01s)
        --- PASS: TestTimeDifferenceGzipPlain/random-line/1.0_MB-gzip-file (0.02s)
        --- PASS: TestTimeDifferenceGzipPlain/random-line/5.2_MB-plain-file (0.04s)
        --- PASS: TestTimeDifferenceGzipPlain/random-line/5.2_MB-gzip-file (0.10s)
        --- PASS: TestTimeDifferenceGzipPlain/random-line/26_MB-plain-file (0.23s)
        --- PASS: TestTimeDifferenceGzipPlain/random-line/26_MB-gzip-file (0.49s)
        --- PASS: TestTimeDifferenceGzipPlain/random-line/52_MB-plain-file (0.44s)
        --- PASS: TestTimeDifferenceGzipPlain/random-line/52_MB-gzip-file (1.00s)
        --- PASS: TestTimeDifferenceGzipPlain/random-line/105_MB-plain-file (1.07s)
        --- PASS: TestTimeDifferenceGzipPlain/random-line/105_MB-gzip-file (2.21s)
        --- PASS: TestTimeDifferenceGzipPlain/random-line/262_MB-plain-file (2.28s)
        --- PASS: TestTimeDifferenceGzipPlain/random-line/262_MB-gzip-file (5.32s)
        --- PASS: TestTimeDifferenceGzipPlain/random-line/524_MB-plain-file (4.18s)
        --- PASS: TestTimeDifferenceGzipPlain/random-line/524_MB-gzip-file (9.79s)
        --- PASS: TestTimeDifferenceGzipPlain/random-line/1.0_GB-plain-file (10.13s)
        --- PASS: TestTimeDifferenceGzipPlain/random-line/1.0_GB-gzip-file (19.56s)
PASS
ok  	github.com/elastic/beats/v7/filebeat/input/filestream	1250.622s
```
