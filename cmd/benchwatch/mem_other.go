//go:build !windows

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// memoryOf gives a process's resident set and its peak, in bytes, where /proc has them.
func memoryOf(pid int) (current, peak uint64, ok bool) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, 0, false
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "VmRSS:":
			current = value * 1024
		case "VmHWM:":
			peak = value * 1024
		}
	}
	return current, peak, current != 0
}
