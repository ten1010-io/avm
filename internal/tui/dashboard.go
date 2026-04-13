package tui

import (
	"fmt"
	"strings"

	"github.com/joonseolee/avm/internal/sriov"
)

type dashboardModel struct {
	iommu   sriov.IOMMUStatus
	devices []sriov.Device
	cursor  int
	err     error
}

func newDashboardModel() dashboardModel {
	return dashboardModel{}
}

func (m dashboardModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("AVM - Advanced VF Manager"))
	b.WriteString("\n\n")

	b.WriteString(m.renderIOMMUStatus())
	b.WriteString("\n")

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

	b.WriteString(helpStyle.Render("  [↑/↓] Navigate  [Enter] Detail  [r] Refresh  [q] Quit"))

	return boxStyle.Render(b.String())
}

func (m dashboardModel) renderIOMMUStatus() string {
	if m.iommu.Enabled {
		method := m.iommu.Method
		if method == "" {
			method = "detected"
		}
		return fmt.Sprintf("  %s  %s\n",
			enabledStyle.Render("IOMMU: ✓ Enabled"),
			dimStyle.Render(fmt.Sprintf("(%s, %d groups)", method, m.iommu.GroupCount)),
		)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s\n", disabledStyle.Render("IOMMU: ✗ Not Enabled")))
	b.WriteString(warningStyle.Render("  ⚠ SR-IOV requires IOMMU. Enable in BIOS:"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("    Intel: VT-d  /  AMD: AMD-Vi"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("    Then add intel_iommu=on to kernel params"))
	b.WriteString("\n")
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
