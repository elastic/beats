package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/elastic/beats/metricbeat/module/goprof/memprof"
)

func main() {
	mode := flag.String("sort", "", "sort order")
	flag.Parse()

	source := flag.Args()[0]
	profile, err := fetchProfile(source)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var sorter func(a, b *memprof.FunctionInfo) bool
	if *mode != "" {
		for i, t := range profile.SampleType {
			if t.Type == *mode {
				sorter = memprof.TopByStat(i)
				break
			}
		}

		if sorter == nil {
			fmt.Println("Invalid sort mode")
			os.Exit(1)
		}
	}

	summary := memprof.SumSamples(profile.Sample)
	for j, t := range profile.SampleType {
		fmt.Printf("%v: %v %v\n", t.Type, summary[j], t.Unit)
	}
	fmt.Println("")

	printFunctions(profile, sorter)
}

func printFunctions(p *memprof.Profile, mode func(a, b *memprof.FunctionInfo) bool) {
	summary := memprof.CollectFunctionStats(p)
	if mode != nil {
		memprof.SortFunctionStatsBy(summary, mode)
	}

	for _, f := range summary {
		fmt.Printf("function %v: %v\n", f.Function.ID, f.Function.Name)
		fmt.Println("  file: ", f.Function.File)
		if len(f.Parents) > 0 {
			fmt.Println("  parents: ")
			for _, parent := range f.Parents {
				fn := parent.Other.Function
				fmt.Printf("    - (id=%v) %v\n", fn.ID, fn.Name)
				if st := parent.StatsTotal; len(st) > 0 {
					fmt.Println("      stats:")
					for i, t := range p.SampleType {
						fmt.Printf("        %v: %v %v\n", t.Type, st[i], t.Unit)
					}
				}
			}
		}
		if len(f.Children) > 0 {
			fmt.Println("  children: ")
			for _, c := range f.Children {
				fn := c.Other.Function
				fmt.Printf("    - (id=%v) %v\n", fn.ID, fn.Name)
				if st := c.StatsTotal; len(st) > 0 {
					fmt.Println("      stats:")
					for i, t := range p.SampleType {
						fmt.Printf("        %v: %v %v\n", t.Type, st[i], t.Unit)
					}
				}
			}
		}

		fmt.Println("  alloc stats:")
		for j, t := range p.SampleType {
			self := int64(0)
			if len(f.StatsSelf) > 0 {
				self = f.StatsSelf[j]
			}
			fmt.Printf("    %v/%v %v %v\n", self, f.StatsTotal[j], t.Type, t.Unit)
		}
		if len(f.SamplesSelf) > 0 {
			fmt.Println("  allocations")
			for _, s := range f.SamplesSelf {
				loc := s.Locations[0]
				fmt.Printf("    line %v (id: %v at 0x%x):\n", loc.Line, loc.ID, loc.Addr)
				for j, t := range p.SampleType {
					fmt.Printf("      %v %v %v\n", s.Values[j], t.Type, t.Unit)
				}
			}
		}

		fmt.Println("")
	}
}

func fetchProfile(source string) (*memprof.Profile, error) {
	in, err := fetchFile(source)
	if err != nil {
		in, err = fetchHTTP(source)
	}

	if err != nil {
		return nil, err
	}

	defer in.Close()

	content, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}

	return memprof.ParseHeap(content)
}

func fetchHTTP(source string) (io.ReadCloser, error) {
	resp, err := httpGet(source, 60*time.Second)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server response: %s", resp.Status)
	}
	return resp.Body, nil
}

func fetchFile(source string) (io.ReadCloser, error) {
	return os.Open(source)
}

// httpGet is a wrapper around http.Get; it is defined as a variable
// so it can be redefined during for testing.
func httpGet(url string, timeout time.Duration) (*http.Response, error) {
	client := &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: timeout + 5*time.Second,
		},
	}
	return client.Get(url)
}
