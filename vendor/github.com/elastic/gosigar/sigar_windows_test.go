package sigar_test

import (
	"math"
	"os"
	"os/user"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	sigar "github.com/elastic/gosigar"
)

var _ = Describe("SigarWindows", func() {
	Describe("Memory", func() {
		It("gets the total memory", func() {
			mem := sigar.Mem{}
			err := mem.Get()

			Ω(err).ShouldNot(HaveOccurred())
			Ω(mem.Total).Should(BeNumerically(">", 0))
		})
	})

	Describe("Disk", func() {
		It("gets the total disk space", func() {
			usage := sigar.FileSystemUsage{}
			err := usage.Get(os.TempDir())

			Ω(err).ShouldNot(HaveOccurred())
			Ω(usage.Total).Should(BeNumerically(">", 0))
		})
	})

	Describe("Process", func() {
		It("gets the current process user name", func() {
			proc := sigar.ProcState{}
			err := proc.Get(os.Getpid())
			user, usererr := user.Current()

			Ω(err).ShouldNot(HaveOccurred())
			Ω(usererr).ShouldNot(HaveOccurred())
			Ω(proc.Username).Should(Equal(user.Username))
		})
	})
})

func TestProcArgs(t *testing.T) {
	args := sigar.ProcArgs{}
	err := args.Get(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}

	if len(args.List) == 0 {
		t.Fatalf("Expected at least one arg")
	}
}

func TestProcArgsUnknown(t *testing.T) {
	args := sigar.ProcArgs{}
	err := args.Get(math.MaxInt32)
	if err == nil {
		t.Fatal("Expected process not found")
	}

	if !strings.Contains(err.Error(), "Process not found") {
		t.Fatal("Expected error containing 'Process not found'")
	}
}
