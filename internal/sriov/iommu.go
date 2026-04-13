package sriov

import (
	"os"
	"os/exec"
	"strings"
)

type IOMMUState int

const (
	IOMMUNotSupported IOMMUState = iota
	IOMMUPassthrough
	IOMMUEnabled
)

type IOMMUStatus struct {
	State       IOMMUState
	Method      string // "intel_iommu=on", "iommu=pt", "Passthrough", etc.
	HasGroups   bool
	GroupCount  int
	DmesgInfo   string // raw dmesg line for display
}

func DetectIOMMU() IOMMUStatus {
	status := IOMMUStatus{}

	status.Method = detectKernelParam()
	status.HasGroups, status.GroupCount = detectIOMMUGroups()
	status.DmesgInfo = detectDmesgIOMMU()

	switch {
	case status.HasGroups && status.GroupCount > 0:
		status.State = IOMMUEnabled
	case status.DmesgInfo != "":
		status.State = IOMMUPassthrough
	default:
		status.State = IOMMUNotSupported
	}

	return status
}

func detectKernelParam() string {
	data, err := os.ReadFile("/proc/cmdline")
	if err != nil {
		return ""
	}

	cmdline := string(data)
	params := []string{"intel_iommu=on", "amd_iommu=on", "iommu=pt"}

	for _, p := range params {
		if strings.Contains(cmdline, p) {
			return p
		}
	}

	return ""
}

func detectIOMMUGroups() (bool, int) {
	entries, err := os.ReadDir("/sys/kernel/iommu_groups")
	if err != nil {
		return false, 0
	}

	count := 0
	for _, e := range entries {
		if e.IsDir() {
			count++
		}
	}

	return count > 0, count
}

func detectDmesgIOMMU() string {
	out, err := exec.Command("dmesg").Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "iommu") {
			return strings.TrimSpace(line)
		}
	}

	return ""
}
