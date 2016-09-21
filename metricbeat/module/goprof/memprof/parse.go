package memprof

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var errUnrecognized = fmt.Errorf("unrecognized profile format")
var errMalformed = fmt.Errorf("malformed profile format")

var (
	heapHeaderRE = regexp.MustCompile(`heap profile: *(\d+): *(\d+) *\[ *(\d+): *(\d+) *\] *@ *(heap[_a-z0-9]*)/?(\d*)`)

	heapSampleRE = regexp.MustCompile(`(-?\d+): *(-?\d+) *\[ *(\d+): *(\d+) *] @([ x0-9a-f]*)`)

	hexNumberRE = regexp.MustCompile(`0x[0-9a-f]+`)

	locationRE = regexp.MustCompile(`#\t(0x[0-9a-f]+)\t(.+)\+0x[0-9a-f]+\t+(.+):(\d+)`)
)

func ParseHeap(b []byte) (*Profile, error) {
	type state struct {
		value []int64
		locs  []*Location
	}

	type rel struct {
		parent, child int
	}

	type fnKey struct {
		Name string
		File string
	}

	r := bytes.NewBuffer(b)

	l, err := r.ReadString('\n')
	if err != nil {
		return nil, errUnrecognized
	}

	l = strings.TrimSpace(l)
	header := heapHeaderRE.FindStringSubmatch(l)
	if header == nil {
		return nil, errUnrecognized
	}

	sampling := ""
	var period int64
	if len(header[6]) > 0 {
		if period, err = strconv.ParseInt(header[6], 10, 64); err != nil {
			return nil, errUnrecognized
		}
	}

	switch header[5] {
	case "heapz_v2", "heap_v2":
		sampling = "v2"
	case "heapprofile":
		sampling, period = "", 1
	case "heap":
		sampling, period = "v2", period/2
	default:
		return nil, errUnrecognized
	}

	locs := map[uint64]*Location{}
	funcs := map[fnKey]*Function{}

	relLocs := map[rel]bool{}
	relFuncs := map[rel]bool{}

	p := &Profile{
		SampleType: []*ValueType{
			{Type: "inuse_objects", Unit: "count"},
			{Type: "inuse_space", Unit: "bytes"},
			{Type: "alloc_objects", Unit: "count"},
			{Type: "alloc_space", Unit: "bytes"},
		},
	}

	// parse heap entries
	var active *state
	locID := -1
	funcID := -1
	finalize := func() {
		if active == nil {
			return
		}

		p.Sample = append(p.Sample, Sample{
			Values:    active.value,
			Locations: active.locs,
		})

		active = nil
		locID = -1
		funcID = -1
	}

	for {
		l, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if !isHeapInfo(l) {
			finalize()
			continue
		}

		if isLocation(l) {
			if active == nil {
				return nil, errMalformed
			}

			addr, fn, file, line, err := parseSampleLocation(l)
			if err != nil {
				return nil, err
			}
			loc := locs[addr]
			var fun *Function
			if loc == nil {
				key := fnKey{
					Name: fn,
					File: file,
				}

				fun = funcs[key]
				if fun == nil {
					fun = &Function{ID: len(funcs), Name: fn, File: file}
					funcs[key] = fun
				}

				loc = &Location{len(locs), addr, fun, line, nil, nil}
				locs[addr] = loc
			} else {
				fun = loc.Function
			}

			if locID >= 0 {
				relLocs[rel{child: locID, parent: loc.ID}] = true
			}
			if funcID >= 0 {
				relFuncs[rel{child: funcID, parent: fun.ID}] = true
			}
			locID = loc.ID
			funcID = fun.ID
			active.locs = append(active.locs, loc)
		} else {
			finalize()

			value, _, _, err := parseHeapSample(l, period, sampling)
			if err != nil {
				return nil, err
			}

			active = &state{value: value}
		}
	}
	finalize()

	// normalize location map
	p.Locations = make([]*Location, len(locs))
	for _, loc := range locs {
		p.Locations[loc.ID] = loc
	}

	// link locations
	for rel := range relLocs {
		parent := p.Locations[rel.parent]
		child := p.Locations[rel.child]
		parent.Children = append(parent.Children, rel.child)
		child.Parents = append(child.Parents, rel.parent)
	}

	// normalize function map
	p.Functions = make([]*Function, len(funcs))
	for _, fn := range funcs {
		p.Functions[fn.ID] = fn
	}

	// link function map
	for rel := range relFuncs {
		parent := p.Functions[rel.parent]
		child := p.Functions[rel.child]
		parent.Children = append(parent.Children, rel.child)
		child.Parents = append(child.Parents, rel.parent)
	}

	return p, nil
}

func isHeapInfo(l string) bool {
	l = strings.TrimSpace(l)
	if len(l) == 0 {
		return false
	}

	return l[0] != '#' || strings.HasPrefix(l, "#\t0x")
}

func isLocation(l string) bool {
	return l[0] == '#'
}

func parseSampleLocation(line string) (uint64, string, string, uint64, error) {
	locData := locationRE.FindStringSubmatch(line)
	if len(locData) != 5 {
		return 0, "", "", 0, fmt.Errorf("unexpected number of location items. got %d, want 5", len(locData))
	}

	addr := parseNumber(locData[1])
	function := strings.TrimSpace(locData[2])
	file := strings.TrimSpace(locData[3])
	ln := parseNumber(locData[4])
	return addr, function, file, ln, nil
}

// parseHeapSample parses a single row from a heap profile into a new Sample.
func parseHeapSample(line string, rate int64, sampling string) (value []int64, blocksize int64, addrs []uint64, err error) {
	sampleData := heapSampleRE.FindStringSubmatch(line)
	if len(sampleData) != 6 {
		return value, blocksize, addrs, fmt.Errorf("unexpected number of sample values: got %d, want 6", len(sampleData))
	}

	// Use first two values by default; tcmalloc sampling generates the
	// same value for both, only the older heap-profile collect separate
	// stats for in-use and allocated objects.
	valueIndex := 1

	var v1, v2, v3, v4 int64
	if v1, err = strconv.ParseInt(sampleData[valueIndex], 10, 64); err != nil {
		return value, blocksize, addrs, fmt.Errorf("malformed sample: %s: %v", line, err)
	}
	if v2, err = strconv.ParseInt(sampleData[valueIndex+1], 10, 64); err != nil {
		return value, blocksize, addrs, fmt.Errorf("malformed sample: %s: %v", line, err)
	}
	if v3, err = strconv.ParseInt(sampleData[valueIndex+2], 10, 64); err != nil {
		return value, blocksize, addrs, fmt.Errorf("malformed sample: %s: %v", line, err)
	}
	if v4, err = strconv.ParseInt(sampleData[valueIndex+3], 10, 64); err != nil {
		return value, blocksize, addrs, fmt.Errorf("malformed sample: %s: %v", line, err)
	}

	if v1 == 0 {
		if v2 != 0 {
			return value, blocksize, addrs, fmt.Errorf("allocation count was 0 but allocation bytes was %d", v2)
		}
	} else {
		blocksize = v2 / v1
		if sampling == "v2" {
			v1, v2 = scaleHeapSample(v1, v2, rate)
			v3, v4 = scaleHeapSample(v3, v4, rate)
		}
	}

	value = []int64{v1, v2, v3, v4}
	addrs = extractHexNumbers(sampleData[5])

	return value, blocksize, addrs, nil
}

// scaleHeapSample adjusts the data from a heapz Sample to
// account for its probability of appearing in the collected
// data. heapz profiles are a sampling of the memory allocations
// requests in a program. We estimate the unsampled value by dividing
// each collected sample by its probability of appearing in the
// profile. heapz v2 profiles rely on a poisson process to determine
// which samples to collect, based on the desired average collection
// rate R. The probability of a sample of size S to appear in that
// profile is 1-exp(-S/R).
func scaleHeapSample(count, size, rate int64) (int64, int64) {
	if count == 0 || size == 0 {
		return 0, 0
	}

	if rate <= 1 {
		// if rate==1 all samples were collected so no adjustment is needed.
		// if rate<1 treat as unknown and skip scaling.
		return count, size
	}

	avgSize := float64(size) / float64(count)
	scale := 1 / (1 - math.Exp(-avgSize/float64(rate)))

	return int64(float64(count) * scale), int64(float64(size) * scale)
}

func parseNumber(s string) uint64 {
	id, err := strconv.ParseUint(s, 0, 64)
	if err != nil {
		// Do not expect any parsing failures due to the regexp matching.
		panic("failed to parse hex value: " + s)
	}
	return id
}

func extractHexNumbers(s string) []uint64 {
	var ids []uint64
	for _, s := range hexNumberRE.FindAllString(s, -1) {
		ids = append(ids, parseNumber(s))
	}
	return ids
}
