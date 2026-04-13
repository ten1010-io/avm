package sriov

import (
	"os"
	"strings"
)

type IOMMUStatus struct {
	Enabled    bool
	Method     string // "intel_iommu", "amd_iommu", "iommu=pt", etc.
	HasGroups  bool
	GroupCount int
}

func DetectIOMMU() IOMMUStatus {
	status := IOMMUStatus{}

	status.Method = detectKernelParam()
	status.HasGroups, status.GroupCount = detectIOMMUGroups()
	status.Enabled = status.HasGroups && status.GroupCount > 0

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
