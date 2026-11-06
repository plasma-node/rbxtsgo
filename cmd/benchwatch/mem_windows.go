package main

import (
	"syscall"
	"unsafe"
)

var (
	psapi                = syscall.NewLazyDLL("psapi.dll")
	getProcessMemoryInfo = psapi.NewProc("GetProcessMemoryInfo")
)

// processMemoryCounters is PROCESS_MEMORY_COUNTERS.
type processMemoryCounters struct {
	size                       uint32
	pageFaultCount             uint32
	peakWorkingSetSize         uintptr
	workingSetSize             uintptr
	quotaPeakPagedPoolUsage    uintptr
	quotaPagedPoolUsage        uintptr
	quotaPeakNonPagedPoolUsage uintptr
	quotaNonPagedPoolUsage     uintptr
	pagefileUsage              uintptr
	peakPagefileUsage          uintptr
}

// memoryOf gives a process's working set and its peak, in bytes.
func memoryOf(pid int) (current, peak uint64, ok bool) {
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION|0x0010, false, uint32(pid))
	if err != nil {
		return 0, 0, false
	}
	defer syscall.CloseHandle(handle)
	var counters processMemoryCounters
	counters.size = uint32(unsafe.Sizeof(counters))
	result, _, _ := getProcessMemoryInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&counters)), uintptr(counters.size))
	if result == 0 {
		return 0, 0, false
	}
	return uint64(counters.workingSetSize), uint64(counters.peakWorkingSetSize), true
}
