package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// An URL like /pattern?r='200x50,404x20,200|500x30'
// The above pattern would return 50 200 responses, then 20 404s, then randomly return a mix of 200 and 500
// responses 30 times

func main() {
	states := &sync.Map{}

	var reqs uint64 = 0

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddUint64(&reqs, 1)

		writer.Write([]byte("Dummy HTTP Server"))
	})

	http.HandleFunc("/pattern", func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddUint64(&reqs, 1)

		status, body := handlePattern(states, request.URL)
		writer.WriteHeader(status)
		writer.Write([]byte(body))
	})

	go func() {
		for {
			time.Sleep(time.Second * 10)
			r := atomic.LoadUint64(&reqs)
			fmt.Printf("Processed %d reqs\n", r)
		}
	}()

	port := 5678
	fmt.Printf("Starting server on port %d\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Printf("Could not start server: %s", err)
		os.Exit(1)
	}
}

type responsePattern struct {
	httpStatuses    []int
	httpStatusesLen int
	countLimit      int
}

func (rp *responsePattern) next() (status int, body string) {
	var idx int
	if rp.httpStatusesLen > 1 {
		fmt.Printf("INTN %d\n", rp.httpStatusesLen)
		idx = rand.Intn(rp.httpStatusesLen)
	} else {
		idx = 0
	}
	status = rp.httpStatuses[idx]
	return status, strconv.Itoa(status)
}

type responsePatternSequence struct {
	currentPatternIdx   int
	currentPattern      *responsePattern
	currentPatternCount int
	patterns            []*responsePattern
	shuffle             bool
	mtx                 sync.Mutex
}

func (ps *responsePatternSequence) next() (status int, body string) {
	ps.mtx.Lock()
	ps.mtx.Unlock()

	if ps.currentPatternCount >= ps.currentPattern.countLimit {
		ps.advancePattern()
	}

	ps.currentPatternCount = ps.currentPatternCount + 1
	return ps.currentPattern.next()
}

func (ps *responsePatternSequence) advancePattern() {
	if ps.shuffle {
		ps.currentPatternIdx = rand.Intn(len(ps.patterns)) - 1
		ps.currentPattern = ps.patterns[ps.currentPatternIdx]
	} else {
		var nextIdx = ps.currentPatternIdx + 1
		if nextIdx == len(ps.patterns) {
			nextIdx = 0
		}
		ps.currentPatternIdx = nextIdx
		ps.currentPattern = ps.patterns[nextIdx]
	}

	ps.currentPatternCount = 0
}

var statusListRegexp = regexp.MustCompile("^[|\\d]+$")

func handlePattern(states *sync.Map, url *url.URL) (status int, body string) {
	query := url.Query()

	rpsInter, ok := states.Load(url.RawQuery)
	var rps *responsePatternSequence
	if !ok {
		patterns, err := compilePatterns(query.Get("r"))
		if err != nil {
			return 400, err.Error()
		}
		rps = NewResponsePatternSequence(patterns, query.Get("shuffle") == "true")
		states.Store(url.RawQuery, rps)
	} else {
		rps = rpsInter.(*responsePatternSequence)
	}

	return rps.next()
}

func NewResponsePatternSequence(patterns []*responsePattern, shuffle bool) *responsePatternSequence {
	ps := responsePatternSequence{
		currentPatternIdx:   0,
		currentPattern:      patterns[0],
		currentPatternCount: 0,
		patterns:            patterns,
		shuffle:             shuffle,
		mtx:                 sync.Mutex{},
	}

	return &ps
}

func compilePatterns(patternsStr string) (patterns []*responsePattern, err error) {
	splitPatterns := strings.Split(patternsStr, ",")

	for _, patternStr := range splitPatterns {
		rp, err := compilePattern(patternStr)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, rp)
	}

	return patterns, nil
}

func compilePattern(patternStr string) (*responsePattern, error) {
	rp := responsePattern{}

	splitPattern := strings.Split(patternStr, "x")
	if len(splitPattern) != 2 {
		return nil, fmt.Errorf("Bad pattern '%s', expected a STATUSxCOUNT as pattern. Got %s")
	}

	statusDefStr := splitPattern[0]
	if statusListRegexp.MatchString(statusDefStr) {
		statuses := strings.Split(statusDefStr, "|")
		for _, statusStr := range statuses {
			status, _ := strconv.Atoi(statusStr)
			rp.httpStatuses = append(rp.httpStatuses, status)
		}
		rp.httpStatusesLen = len(rp.httpStatuses)
	} else {
		return nil, fmt.Errorf("Expected a | separated list of numbers for status code def, got '%s'", statusDefStr)

	}

	count, err := strconv.Atoi(splitPattern[1])
	if err != nil {
		return nil, fmt.Errorf("Repeat def should be an int, got '%s'", splitPattern[1])
	}
	rp.countLimit = count

	return &rp, nil
}
