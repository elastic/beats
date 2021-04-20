// +build linux windows

package platformcheck

import (
	"fmt"
	"math/bits"
	"strings"

	"github.com/shirou/gopsutil/host"
)

func CheckNativePlatformCompat() error {
	const compiledArchBits = bits.UintSize // 32 if the binary was compiled for 32 bit architecture.

	if compiledArchBits > 32 {
		// We assume that 64bit binaries can only be run on 64bit systems
		return nil
	}

	arch, err := host.KernelArch()
	if err != nil {
		return err
	}

	if strings.Contains(arch, "64") {
		return fmt.Errorf("trying to run %vBit binary on 64Bit system", compiledArchBits)
	}

	return nil
}
