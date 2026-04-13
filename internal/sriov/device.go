package sriov

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Device struct {
	BDF       string // Bus:Device.Function (e.g., "0000:03:00.0")
	VendorID  string // e.g., "8086"
	DeviceID  string // e.g., "1572"
	Vendor    string // e.g., "Intel Corporation"
	DevName   string // e.g., "Ethernet Controller X710"
	Driver    string // e.g., "i40e"
	Class     string // e.g., "Ethernet controller"
	TotalVFs  int
	NumVFs    int
	NetIfaces []string // associated network interfaces
}

func ScanDevices() ([]Device, error) {
	pattern := "/sys/bus/pci/devices/*/sriov_totalvfs"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to scan PCI devices: %w", err)
	}

	devices := make([]Device, 0, len(matches))
	for _, match := range matches {
		devPath := filepath.Dir(match)
		bdf := filepath.Base(devPath)

		dev := Device{BDF: bdf}
		dev.VendorID = readSysfsField(devPath, "vendor")
		dev.DeviceID = readSysfsField(devPath, "device")
		dev.TotalVFs = readSysfsInt(devPath, "sriov_totalvfs")
		dev.NumVFs = readSysfsInt(devPath, "sriov_numvfs")
		dev.Driver = readDriverName(devPath)
		dev.NetIfaces = readNetInterfaces(devPath)

		vendor, devName, class := lookupDeviceInfo(bdf)
		dev.Vendor = vendor
		dev.DevName = devName
		dev.Class = class

		devices = append(devices, dev)
	}

	return devices, nil
}

func readSysfsField(devPath, field string) string {
	data, err := os.ReadFile(filepath.Join(devPath, field))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(string(data), "0x"))
}

func readSysfsInt(devPath, field string) int {
	data, err := os.ReadFile(filepath.Join(devPath, field))
	if err != nil {
		return 0
	}
	val, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return val
}

func readDriverName(devPath string) string {
	link, err := os.Readlink(filepath.Join(devPath, "driver"))
	if err != nil {
		return "none"
	}
	return filepath.Base(link)
}

func readNetInterfaces(devPath string) []string {
	netPath := filepath.Join(devPath, "net")
	entries, err := os.ReadDir(netPath)
	if err != nil {
		return nil
	}

	ifaces := make([]string, 0, len(entries))
	for _, e := range entries {
		ifaces = append(ifaces, e.Name())
	}
	return ifaces
}

func lookupDeviceInfo(bdf string) (vendor, device, class string) {
	out, err := exec.Command("lspci", "-vmm", "-s", bdf).Output()
	if err != nil {
		return "Unknown", "Unknown", "Unknown"
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		val := strings.TrimSpace(parts[1])

		switch key {
		case "Vendor":
			vendor = val
		case "Device":
			device = val
		case "Class":
			class = val
		}
	}

	if vendor == "" {
		vendor = "Unknown"
	}
	if device == "" {
		device = "Unknown"
	}
	if class == "" {
		class = "Unknown"
	}

	return vendor, device, class
}
