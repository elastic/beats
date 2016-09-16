package memprof

import "sort"

type FunctionInfo struct {
	Function    *Function
	StatsTotal  []int64
	StatsSelf   []int64
	SamplesAll  []Sample
	SamplesSelf []Sample

	Parents  []RelFunctionInfo
	Children []RelFunctionInfo
}

type RelFunctionInfo struct {
	Other         *FunctionInfo
	StatsDirect   []int64
	StatsIndirect []int64
	StatsTotal    []int64
}

type Sorter struct {
	fis  []*FunctionInfo
	less func(a, b *FunctionInfo) bool
}

func CollectFunctionStats(p *Profile) []*FunctionInfo {
	out := make([]*FunctionInfo, len(p.Functions))
	for i, f := range p.Functions {
		samplesAll := CollectFunctionSamples(f, p.Sample, false)
		samplesSelf := CollectFunctionSamples(f, samplesAll, true)
		out[i] = &FunctionInfo{
			Function:    f,
			StatsTotal:  SumSamples(samplesAll),
			StatsSelf:   SumSamples(samplesSelf),
			SamplesAll:  samplesAll,
			SamplesSelf: samplesSelf,
			Parents:     make([]RelFunctionInfo, len(f.Parents)),
			Children:    make([]RelFunctionInfo, len(f.Children)),
		}
	}

	// link function from leaf -> root
	for _, fi := range out {
		for i, id := range fi.Function.Parents {
			parent := out[id]
			statsDirect := SumSamples(CollectFunctionSamples(
				parent.Function, fi.SamplesSelf, false))

			statsIndirect := SumSamples(CollectFunctionSamples(
				parent.Function,
				CollectNonLeafFunctionSamples(fi.Function, fi.SamplesAll),
				false))

			fi.Parents[i] = RelFunctionInfo{
				Other:         parent,
				StatsDirect:   statsDirect,
				StatsIndirect: statsIndirect,
				StatsTotal:    SumValues(statsIndirect, statsDirect),
			}
		}

	}

	// link function from root -> leaf, copying already computed stats
	for _, fi := range out {
		for i, id := range fi.Function.Children {
			child := out[id]
			var rel *RelFunctionInfo
			for i := range child.Parents {
				r := &child.Parents[i]
				if r.Other == fi {
					rel = r
					break
				}
			}
			if rel == nil {
				panic("missing link")
			}

			fi.Children[i] = RelFunctionInfo{
				Other:         child,
				StatsDirect:   rel.StatsDirect,
				StatsIndirect: rel.StatsIndirect,
				StatsTotal:    rel.StatsTotal,
			}
		}
	}

	// summarize per function stats
	for _, fi := range out {
		fi.SamplesSelf = CombineSamplesByLocation(fi.SamplesSelf)
		fi.SamplesAll = CombineSamplesByLocation(fi.SamplesAll)
	}

	return out
}

func (s *Sorter) Len() int           { return len(s.fis) }
func (s *Sorter) Swap(i, j int)      { s.fis[i], s.fis[j] = s.fis[j], s.fis[i] }
func (s *Sorter) Less(i, j int) bool { return s.less(s.fis[i], s.fis[j]) }

func SortFunctionStatsBy(fis []*FunctionInfo, cmp func(a, b *FunctionInfo) bool) {
	sort.Sort(&Sorter{fis, cmp})
}

func TopByStat(idx int) func(*FunctionInfo, *FunctionInfo) bool {
	return func(b, a *FunctionInfo) bool {
		ca := len(a.StatsSelf)
		cb := len(b.StatsSelf)
		if ca != cb {
			return ca == 0
		}

		sa := a.StatsSelf
		sb := b.StatsSelf
		if ca == 0 {
			sa = a.StatsTotal
			sb = b.StatsTotal
		}
		return sa[idx] < sb[idx]
	}
}
