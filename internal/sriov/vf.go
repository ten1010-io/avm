package sriov

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type VFInfo struct {
	Index    int
	BDF      string // e.g., "0000:03:02.0"
	Driver   string // e.g., "vfio-pci", "i40e", "none"
	MAC      string
	LinkState string // "auto", "enable", "disable"
	// K8s info (populated separately)
	PodName     string
	PodNS       string
	NetworkName string
	MinTxRate   int // Mbps, from SriovNetwork
	MaxTxRate   int // Mbps, from SriovNetwork
	VLAN        int
	SpoofChk    string
	Trust       string
}

func ScanVFs(pfBDF string) ([]VFInfo, error) {
	devPath := filepath.Join("/sys/bus/pci/devices", pfBDF)

	numVFs := readSysfsInt(devPath, "sriov_numvfs")
	if numVFs == 0 {
		return nil, nil
	}

	vfs := make([]VFInfo, 0, numVFs)
	for i := 0; i < numVFs; i++ {
		vfLink := filepath.Join(devPath, fmt.Sprintf("virtfn%d", i))
		realPath, err := filepath.EvalSymlinks(vfLink)
		if err != nil {
			continue
		}

		vfBDF := filepath.Base(realPath)
		vf := VFInfo{
			Index:  i,
			BDF:    vfBDF,
			Driver: readDriverName(realPath),
			MAC:    readVFMAC(pfBDF, i),
		}

		vfs = append(vfs, vf)
	}

	return vfs, nil
}

func readVFMAC(pfBDF string, vfIndex int) string {
	devPath := filepath.Join("/sys/bus/pci/devices", pfBDF)

	netEntries, err := os.ReadDir(filepath.Join(devPath, "net"))
	if err != nil || len(netEntries) == 0 {
		return ""
	}

	ifName := netEntries[0].Name()
	// Read from /sys/class/net/<ifname>/device/sriov/<vfIndex>/address
	addrPath := filepath.Join("/sys/class/net", ifName, "device", "sriov", fmt.Sprintf("%d", vfIndex), "address")
	data, err := os.ReadFile(addrPath)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

func GetVFCount(bdf string) (current int, max int, err error) {
	devPath := filepath.Join("/sys/bus/pci/devices", bdf)

	maxData, err := os.ReadFile(filepath.Join(devPath, "sriov_totalvfs"))
	if err != nil {
		return 0, 0, fmt.Errorf("cannot read max VFs for %s: %w", bdf, err)
	}
	max, err = strconv.Atoi(strings.TrimSpace(string(maxData)))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid max VF value for %s: %w", bdf, err)
	}

	curData, err := os.ReadFile(filepath.Join(devPath, "sriov_numvfs"))
	if err != nil {
		return 0, max, fmt.Errorf("cannot read current VFs for %s: %w", bdf, err)
	}
	current, err = strconv.Atoi(strings.TrimSpace(string(curData)))
	if err != nil {
		return 0, max, fmt.Errorf("invalid current VF value for %s: %w", bdf, err)
	}

	return current, max, nil
}

func SetVFCount(bdf string, count int) error {
	devPath := filepath.Join("/sys/bus/pci/devices", bdf)

	maxData, err := os.ReadFile(filepath.Join(devPath, "sriov_totalvfs"))
	if err != nil {
		return fmt.Errorf("cannot read max VFs for %s: %w", bdf, err)
	}
	maxVFs, err := strconv.Atoi(strings.TrimSpace(string(maxData)))
	if err != nil {
		return fmt.Errorf("invalid max VF value for %s: %w", bdf, err)
	}

	if count < 0 || count > maxVFs {
		return fmt.Errorf("VF count %d out of range [0, %d] for %s", count, maxVFs, bdf)
	}

	numvfsPath := filepath.Join(devPath, "sriov_numvfs")
	err = os.WriteFile(numvfsPath, []byte(strconv.Itoa(count)), 0644)
	if err != nil {
		return fmt.Errorf("failed to set VF count for %s (root required): %w", bdf, err)
	}

	return nil
}
