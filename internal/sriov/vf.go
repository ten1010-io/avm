package sriov

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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
