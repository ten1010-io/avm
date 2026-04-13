package sriov

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type PKeyPartition struct {
	PKey    string // e.g., "0x8001"
	Name    string // comment-based name (parsed from config)
	Members string // e.g., "ALL=full", "0xGUID1=full, 0xGUID2=limited"
	Active  bool
}

const defaultPartitionsConf = "/etc/opensm/partitions.conf"

func DetectSubnetManager() (name string, active bool) {
	// Check if opensm is running
	out, err := exec.Command("systemctl", "is-active", "opensm").Output()
	if err == nil && strings.TrimSpace(string(out)) == "active" {
		return "OpenSM", true
	}

	// Check process list as fallback
	out, err = exec.Command("pgrep", "-x", "opensm").Output()
	if err == nil && len(strings.TrimSpace(string(out))) > 0 {
		return "OpenSM", true
	}

	return "OpenSM", false
}

func FindPartitionsConf() string {
	paths := []string{
		"/etc/opensm/partitions.conf",
		"/etc/opensm/partitions.policy",
		"/usr/local/etc/opensm/partitions.conf",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

func ReadPKeyPartitions() ([]PKeyPartition, string, error) {
	confPath := FindPartitionsConf()
	if confPath == "" {
		return nil, "", fmt.Errorf("partitions.conf not found")
	}

	data, err := os.ReadFile(confPath)
	if err != nil {
		return nil, confPath, fmt.Errorf("cannot read %s: %w", confPath, err)
	}

	partitions := parsePKeyPartitions(string(data))

	// Check which P-Keys are actually active via sysfs
	enrichWithActiveStatus(partitions)

	return partitions, confPath, nil
}

func parsePKeyPartitions(content string) []PKeyPartition {
	var partitions []PKeyPartition
	var currentName string

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Comment lines might contain partition names
		if strings.HasPrefix(line, "#") {
			trimmed := strings.TrimSpace(strings.TrimPrefix(line, "#"))
			if trimmed != "" && !strings.HasPrefix(trimmed, "!") {
				currentName = trimmed
			}
			continue
		}

		if line == "" {
			continue
		}

		// Format: pkey_value : member_list ;
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		pkey := strings.TrimSpace(parts[0])
		members := strings.TrimSuffix(strings.TrimSpace(parts[1]), ";")
		members = strings.TrimSpace(members)

		partition := PKeyPartition{
			PKey:    normalizePKey(pkey),
			Name:    currentName,
			Members: members,
		}

		partitions = append(partitions, partition)
		currentName = ""
	}

	return partitions
}

func enrichWithActiveStatus(partitions []PKeyPartition) {
	// Read active P-Keys from sysfs
	activePKeys := readActivePKeys()

	for i := range partitions {
		_, partitions[i].Active = activePKeys[partitions[i].PKey]
	}
}

func readActivePKeys() map[string]bool {
	active := make(map[string]bool)

	// Check /sys/class/infiniband/*/ports/*/pkeys/*
	matches, err := filepath.Glob("/sys/class/infiniband/*/ports/*/pkeys/*")
	if err != nil {
		return active
	}

	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			continue
		}

		val := strings.TrimSpace(string(data))
		if val != "0x0000" && val != "0x0" {
			active[normalizePKey(val)] = true
		}
	}

	return active
}

func normalizePKey(pkey string) string {
	pkey = strings.TrimSpace(pkey)
	pkey = strings.TrimPrefix(pkey, "0x")
	pkey = strings.TrimPrefix(pkey, "0X")

	val, err := strconv.ParseUint(pkey, 16, 16)
	if err != nil {
		return "0x" + pkey
	}

	return fmt.Sprintf("0x%04X", val)
}

func AddPKeyPartition(name string, pkey string, members string) error {
	confPath := FindPartitionsConf()
	if confPath == "" {
		confPath = defaultPartitionsConf
	}

	// Backup
	backupPath := confPath + ".bak.avm"
	existing, _ := os.ReadFile(confPath)
	if len(existing) > 0 {
		if err := os.WriteFile(backupPath, existing, 0644); err != nil {
			return fmt.Errorf("cannot create backup: %w", err)
		}
	}

	normalizedPKey := normalizePKey(pkey)

	// Check for duplicate
	if existing != nil && strings.Contains(string(existing), normalizedPKey) {
		return fmt.Errorf("P-Key %s already exists in config", normalizedPKey)
	}

	if members == "" {
		members = "ALL=full"
	}

	entry := fmt.Sprintf("\n# %s\n%s : %s ;\n", name, normalizedPKey, members)

	f, err := os.OpenFile(confPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("cannot write to %s: %w", confPath, err)
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	// Restart opensm to apply
	if err := restartOpenSM(); err != nil {
		return fmt.Errorf("config written but opensm restart failed: %w", err)
	}

	return nil
}

func DeletePKeyPartition(pkey string) error {
	confPath := FindPartitionsConf()
	if confPath == "" {
		return fmt.Errorf("partitions.conf not found")
	}

	data, err := os.ReadFile(confPath)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", confPath, err)
	}

	// Backup
	backupPath := confPath + ".bak.avm"
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("cannot create backup: %w", err)
	}

	normalizedPKey := normalizePKey(pkey)
	lines := strings.Split(string(data), "\n")
	var result []string
	skipNext := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// If previous line was a comment for this partition, it was already skipped
		if skipNext {
			if strings.Contains(trimmed, normalizedPKey) {
				skipNext = false
				continue // Skip the pkey line
			}
			// Comment wasn't for this pkey, keep it
			skipNext = false
		}

		// Check if this line defines the target pkey
		if !strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, normalizedPKey) {
			// Check if previous line was a comment (name)
			if len(result) > 0 {
				prevLine := strings.TrimSpace(result[len(result)-1])
				if strings.HasPrefix(prevLine, "#") {
					result = result[:len(result)-1] // Remove associated comment
				}
			}
			continue // Skip this pkey line
		}

		result = append(result, line)
	}

	if err := os.WriteFile(confPath, []byte(strings.Join(result, "\n")), 0644); err != nil {
		return fmt.Errorf("cannot write %s: %w", confPath, err)
	}

	if err := restartOpenSM(); err != nil {
		return fmt.Errorf("config updated but opensm restart failed: %w", err)
	}

	return nil
}

func restartOpenSM() error {
	if err := exec.Command("systemctl", "restart", "opensm").Run(); err != nil {
		// Fallback: try sending SIGHUP
		if err2 := exec.Command("pkill", "-HUP", "opensm").Run(); err2 != nil {
			return fmt.Errorf("could not restart opensm: %w", err)
		}
	}
	return nil
}
