package memprof

type Profile struct {
	SampleType []*ValueType
	Sample     []Sample
	Locations  []*Location
	Functions  []*Function
}

type ValueType struct {
	Type string
	Unit string
}

type Sample struct {
	Values    []int64
	Locations []*Location
}

type Location struct {
	ID       int
	Addr     uint64
	Function *Function
	Line     uint64

	// parent/children location ids
	Parents  []int
	Children []int
}

type Function struct {
	ID   int
	Name string
	File string

	// parent/children location ids
	Parents  []int
	Children []int
}

func CollectLocationSamples(loc *Location, samples []Sample, leafOnly bool) []Sample {
	if leafOnly {
		return CollectSamplesIf(samples, loc.IsSampleLeaf)
	}
	return CollectSamplesIf(samples, loc.HasSample)
}

func CollectFunctionSamples(f *Function, samples []Sample, leafOnly bool) []Sample {
	if leafOnly {
		return CollectSamplesIf(samples, f.IsSampleLeaf)
	}
	return CollectSamplesIf(samples, f.HasSample)
}

func CollectNonLeafFunctionSamples(f *Function, samples []Sample) []Sample {
	return CollectSamplesIf(samples, func(s *Sample) bool { return !f.IsSampleLeaf(s) })
}

func CollectSamplesIf(samples []Sample, check func(s *Sample) bool) []Sample {
	var out []Sample

	for _, s := range samples {
		if check(&s) {
			out = append(out, s)
		}
	}

	return out
}

func SumSamples(samples []Sample) []int64 {
	if len(samples) == 0 {
		return nil
	}

	values := make([]int64, len(samples[0].Values))
	for _, s := range samples {
		for i, v := range s.Values {
			values[i] += v
		}
	}
	return values
}

func SumValues(vs ...[]int64) []int64 {
	if len(vs) == 0 {
		return nil
	}

	L := 0
	for _, s := range vs {
		if len(s) > L {
			L = len(s)
		}
	}

	if L == 0 {
		return nil
	}

	values := make([]int64, L)
	for _, s := range vs {
		for i, v := range s {
			values[i] += v
		}
	}
	return values
}

func SubValues(a, b []int64) []int64 {
	L := len(a)
	if L == 0 {
		return nil
	}

	if lb := len(b); L < lb {
		L = lb
	}

	values := make([]int64, L)
	copy(values, a)
	for i := range b {
		values[i] -= b[i]
	}
	return values
}

func CombineSamplesByLocation(samples []Sample) []Sample {
	// return if no duplicate entries
	if len(samples) <= 1 {
		return samples
	}

	// combine multiple samples by same leaf location
	m := map[int]Sample{}
	for _, s := range samples {
		l := s.Locations[0]
		id := l.ID
		if old, exists := m[id]; exists {
			m[id] = Sample{
				Values:    SumSamples([]Sample{old, s}),
				Locations: []*Location{l},
			}
		} else {
			m[id] = s
		}
	}

	// return original samples if no duplicates were found
	if len(m) == len(samples) {
		return samples
	}

	// returned combined samples list
	out := make([]Sample, 0, len(m))
	for _, s := range m {
		out = append(out, s)
	}
	return out
}

func (l *Location) HasSample(s *Sample) bool {
	for _, sl := range s.Locations {
		if sl.Equals(l) {
			return true
		}
	}
	return false
}

func (l *Location) IsSampleLeaf(s *Sample) bool {
	return len(s.Locations) > 0 && s.Locations[0].Equals(l)
}

func (l *Location) InFunction(f *Function) bool {
	return l.Function != nil && l.Function.Equals(f)
}

func (l *Location) Equals(o *Location) bool {
	return l.Addr == o.Addr
}

func (l *Location) IsRoot() bool {
	return len(l.Parents) == 0
}

func (l *Location) IsLeaf() bool {
	return len(l.Children) == 0
}

func (f *Function) HasSample(s *Sample) bool {
	for _, sl := range s.Locations {
		if sl.InFunction(f) {
			return true
		}
	}
	return false
}

func (f *Function) IsSampleLeaf(s *Sample) bool {
	return len(s.Locations) > 0 && s.Locations[0].InFunction(f)
}

func (f *Function) Equals(o *Function) bool {
	return f.File == o.File && f.Name == o.Name
}

func (f *Function) IsRoot() bool {
	return len(f.Parents) == 0
}

func (f *Function) IsLeaf() bool {
	return len(f.Children) == 0
}
