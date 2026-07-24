#!/bin/bash

set -euo pipefail

source .buildkite/scripts/common.sh

install_go_dependencies

# Try to find and print relevant kernel config options
# Searches known locations, returns on first success
print_kernel_config() {
	local pattern='CONFIG_(ZSWAP|SWAP|CGROUP|MEMCG|PSI)'

	# Try /proc/config.gz (requires CONFIG_IKCONFIG_PROC)
	if [ -f /proc/config.gz ]; then
		echo "Using: /proc/config.gz"
		zcat /proc/config.gz 2>/dev/null | grep -E "$pattern" | grep -v '^#' && return 0
	fi

	# Try /boot/config-$(uname -r) (most common on installed systems)
	local boot_config="/boot/config-$(uname -r)"
	if [ -f "$boot_config" ]; then
		echo "Using: $boot_config"
		grep -E "$pattern" "$boot_config" 2>/dev/null | grep -v '^#' && return 0
	fi

	# Try /lib/modules/$(uname -r)/config (some distros)
	local modules_config="/lib/modules/$(uname -r)/config"
	if [ -f "$modules_config" ]; then
		echo "Using: $modules_config"
		grep -E "$pattern" "$modules_config" 2>/dev/null | grep -v '^#' && return 0
	fi

	echo "Kernel config not found in any known location"
}

print_debug_info() {
	echo "=== System Info ==="
	uname -a

	echo "=== Memory Info ==="
	grep -iE 'mem|swap|zswap|cgroup' /proc/meminfo || true

	echo "=== Zswap Module Status ==="
	cat /sys/module/zswap/parameters/enabled 2>/dev/null || echo "zswap module not loaded"

	echo "=== Zswap Debugfs ==="
	sudo grep -r . /sys/kernel/debug/zswap/ 2>/dev/null || echo "debugfs zswap not accessible"

	echo "=== Kernel Config ==="
	print_kernel_config

	echo "=== Mounts (cgroup/debugfs) ==="
	mount | grep -E 'cgroup|debugfs' || echo "no cgroup/debugfs mounts found"
}

print_debug_info || true

go test -timeout 20m -v ./tests
