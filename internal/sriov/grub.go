package sriov

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type GrubUpdateResult struct {
	Success     bool
	BackupPath  string
	NeedReboot  bool
	ErrorMsg    string
}

func EnableIOMMUKernelParam() GrubUpdateResult {
	if runtime.GOOS != "linux" {
		return GrubUpdateResult{ErrorMsg: "only supported on Linux"}
	}

	grubPath := findGrubDefault()
	if grubPath == "" {
		return GrubUpdateResult{ErrorMsg: "cannot find /etc/default/grub"}
	}

	data, err := os.ReadFile(grubPath)
	if err != nil {
		return GrubUpdateResult{ErrorMsg: fmt.Sprintf("cannot read %s: %v", grubPath, err)}
	}

	content := string(data)

	if strings.Contains(content, "intel_iommu=on") {
		return GrubUpdateResult{ErrorMsg: "intel_iommu=on already present in grub config"}
	}

	backupPath := grubPath + ".bak.avm"
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return GrubUpdateResult{ErrorMsg: fmt.Sprintf("cannot create backup: %v", err)}
	}

	updated := addGrubParams(content, "intel_iommu=on iommu=pt")

	if err := os.WriteFile(grubPath, []byte(updated), 0644); err != nil {
		return GrubUpdateResult{
			ErrorMsg:   fmt.Sprintf("cannot write %s: %v", grubPath, err),
			BackupPath: backupPath,
		}
	}

	if err := regenerateGrub(); err != nil {
		return GrubUpdateResult{
			ErrorMsg:   fmt.Sprintf("grub regeneration failed: %v (backup at %s)", err, backupPath),
			BackupPath: backupPath,
		}
	}

	return GrubUpdateResult{
		Success:    true,
		BackupPath: backupPath,
		NeedReboot: true,
	}
}

func ReadCurrentGrubCmdline() string {
	grubPath := findGrubDefault()
	if grubPath == "" {
		return ""
	}

	data, err := os.ReadFile(grubPath)
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "GRUB_CMDLINE_LINUX=") {
			return trimmed
		}
	}

	return ""
}

func findGrubDefault() string {
	path := "/etc/default/grub"
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

func addGrubParams(content, params string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "GRUB_CMDLINE_LINUX=") {
			// Remove trailing quote, add params, re-add quote
			line = strings.TrimRight(trimmed, "\"")
			line = line + " " + params + "\""
			lines[i] = line
			break
		}
	}
	return strings.Join(lines, "\n")
}

func regenerateGrub() error {
	// Try grub2-mkconfig (CentOS/RHEL)
	grubCfgPaths := []string{
		"/boot/grub2/grub.cfg",
		"/boot/efi/EFI/centos/grub.cfg",
		"/boot/efi/EFI/redhat/grub.cfg",
		"/boot/efi/EFI/rocky/grub.cfg",
	}

	for _, cfgPath := range grubCfgPaths {
		if _, err := os.Stat(cfgPath); err == nil {
			return exec.Command("grub2-mkconfig", "-o", cfgPath).Run()
		}
	}

	// Try update-grub (Ubuntu/Debian)
	if path, err := exec.LookPath("update-grub"); err == nil {
		return exec.Command(path).Run()
	}

	// Fallback: grub-mkconfig
	if path, err := exec.LookPath("grub-mkconfig"); err == nil {
		return exec.Command(path, "-o", "/boot/grub/grub.cfg").Run()
	}

	return fmt.Errorf("no grub config tool found (tried grub2-mkconfig, update-grub, grub-mkconfig)")
}
