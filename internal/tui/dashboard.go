package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/joonseolee/avm/internal/sriov"
)

type dashboardModel struct {
	iommu        sriov.IOMMUStatus
	devices      []sriov.Device
	cursor       int
	err          error
	confirmIOMMU bool   // "e" 누르면 true → "y"로 실행
	grubMessage  string // GRUB 수정 결과 메시지
	grubIsError  bool
	lastRefresh  time.Time
}

func newDashboardModel() dashboardModel {
	return dashboardModel{}
}

func (m dashboardModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("AVM - Advanced Virtualization Manager"))
	if !m.lastRefresh.IsZero() {
		b.WriteString("  " + dimStyle.Render(fmt.Sprintf("Last scanned: %s", m.lastRefresh.Format("15:04:05"))))
	}
	b.WriteString("\n\n")

	b.WriteString(m.renderIOMMUStatus())
	b.WriteString("\n")

	if m.grubMessage != "" {
		if m.grubIsError {
			b.WriteString(errorStyle.Render("  "+m.grubMessage) + "\n\n")
		} else {
			b.WriteString(successStyle.Render("  "+m.grubMessage) + "\n\n")
		}
	}

	if m.confirmIOMMU {
		b.WriteString(warningStyle.Render("  ⚠ This will modify /etc/default/grub and regenerate grub config."))
		b.WriteString("\n")
		b.WriteString(warningStyle.Render("    A backup will be saved. Reboot required after."))
		b.WriteString("\n\n")

		currentGrub := sriov.ReadCurrentGrubCmdline()
		if currentGrub != "" {
			b.WriteString(dimStyle.Render("  Current: "+currentGrub) + "\n")
			b.WriteString(dimStyle.Render("  Will add: intel_iommu=on iommu=pt") + "\n\n")
		}

		b.WriteString(headerStyle.Render("  Proceed? [y] Yes  [n] Cancel"))
		b.WriteString("\n")
		return boxStyle.Render(b.String())
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %s", m.err.Error())))
		b.WriteString("\n")
	}

	if len(m.devices) == 0 {
		b.WriteString(dimStyle.Render("  No SR-IOV capable devices found."))
		b.WriteString("\n")
	} else {
		b.WriteString(m.renderDeviceTable())
	}

	help := "  [↑/↓] Navigate  [Enter] Detail  [p] P-Keys  [r] Refresh  [q] Quit"
	if m.iommu.State == sriov.IOMMUPassthrough {
		help = "  [↑/↓] Navigate  [Enter] Detail  [p] P-Keys  [e] Enable IOMMU  [r] Refresh  [q] Quit"
	}
	b.WriteString(helpStyle.Render(help))

	return boxStyle.Render(b.String())
}

func (m dashboardModel) renderIOMMUStatus() string {
	var b strings.Builder

	switch m.iommu.State {
	case sriov.IOMMUEnabled:
		method := m.iommu.Method
		if method == "" {
			method = "detected"
		}
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			enabledStyle.Render("IOMMU: ✓ Enabled"),
			dimStyle.Render(fmt.Sprintf("(%s, %d groups)", method, m.iommu.GroupCount)),
		))

	case sriov.IOMMUPassthrough:
		b.WriteString(fmt.Sprintf("  %s\n", warningStyle.Render("IOMMU: ⚡ Passthrough Mode")))
		b.WriteString(dimStyle.Render("    HW supported but running in passthrough mode"))
		b.WriteString("\n")
		if m.iommu.DmesgInfo != "" {
			b.WriteString(dimStyle.Render(fmt.Sprintf("    dmesg: %s", m.iommu.DmesgInfo)))
			b.WriteString("\n")
		}

	default:
		b.WriteString(fmt.Sprintf("  %s\n", disabledStyle.Render("IOMMU: ✗ Not Supported")))
		b.WriteString(warningStyle.Render("  ⚠ SR-IOV requires IOMMU. Enable in BIOS:"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("    Intel: VT-d  /  AMD: AMD-Vi"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("    Then add intel_iommu=on to kernel params"))
		b.WriteString("\n")
	}

	return b.String()
}

func (m dashboardModel) renderDeviceTable() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("  SR-IOV Capable Devices:"))
	b.WriteString("\n\n")

	header := fmt.Sprintf("  %-14s %-30s %-12s %5s %5s",
		"BDF", "Device", "Driver", "VFs", "Max")
	b.WriteString(dimStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  " + strings.Repeat("─", 70)))
	b.WriteString("\n")

	for i, dev := range m.devices {
		name := dev.DevName
		if len(name) > 28 {
			name = name[:28] + ".."
		}

		row := fmt.Sprintf("  %-14s %-30s %-12s %5d %5d",
			dev.BDF, name, dev.Driver, dev.NumVFs, dev.TotalVFs)

		if i == m.cursor {
			b.WriteString(selectedRowStyle.Render("▸ " + row[2:]))
		} else {
			b.WriteString(normalRowStyle.Render(row))
		}
		b.WriteString("\n")
	}

	return b.String()
}
