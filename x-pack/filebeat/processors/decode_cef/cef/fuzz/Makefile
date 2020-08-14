fuzz:
	go test ../. -corpus
	go get github.com/dvyukov/go-fuzz/go-fuzz github.com/dvyukov/go-fuzz/go-fuzz-build
	go-fuzz-build
	go-fuzz

.PHONY: fuzz
