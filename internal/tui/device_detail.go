package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/joonseolee/avm/internal/sriov"
)

type detailModel struct {
	device   sriov.Device
	vfs      []sriov.VFInfo
	input    textinput.Model
	message  string
	isError  bool
	demoMode bool
	vfScroll int // scroll offset for VF list
}

func newDetailModel(dev sriov.Device, demoMode bool) detailModel {
	ti := textinput.New()
	ti.Placeholder = fmt.Sprintf("0-%d", dev.TotalVFs)
	ti.Focus()
	ti.CharLimit = 4
	ti.Width = 10
	ti.SetValue(fmt.Sprintf("%d", dev.NumVFs))

	var vfs []sriov.VFInfo
	if demoMode {
		vfs = sriov.DemoVFs(dev.BDF)
	} else {
		vfs, _ = sriov.ScanVFs(dev.BDF)
	}

	return detailModel{
		device:   dev,
		vfs:      vfs,
		input:    ti,
		demoMode: demoMode,
	}
}

func (m detailModel) View() string {
	var b strings.Builder

	title := fmt.Sprintf("Device: %s - %s", m.device.BDF, m.device.DevName)
	if len(title) > 60 {
		title = title[:60] + ".."
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	b.WriteString(m.renderDeviceInfo())
	b.WriteString("\n")
	b.WriteString(m.renderVFConfig())

	if len(m.vfs) > 0 {
		b.WriteString("\n")
		b.WriteString(m.renderVFTable())
	}

	if m.demoMode {
		b.WriteString("\n")
		b.WriteString(warningStyle.Render("  [Demo Mode] VF changes are simulated"))
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("  [Enter] Apply VF Count  [Esc] Back"))

	return boxStyle.Render(b.String())
}

func (m detailModel) renderDeviceInfo() string {
	var b strings.Builder

	fields := []struct{ label, value string }{
		{"Vendor", m.device.Vendor},
		{"Device ID", fmt.Sprintf("%s:%s", m.device.VendorID, m.device.DeviceID)},
		{"Driver", m.device.Driver},
		{"Class", m.device.Class},
	}

	if len(m.device.NetIfaces) > 0 {
		fields = append(fields, struct{ label, value string }{
			"Interfaces", strings.Join(m.device.NetIfaces, ", "),
		})
	}

	for _, f := range fields {
		b.WriteString(fmt.Sprintf("  %s %s\n",
			labelStyle.Render(f.label+":"),
			valueStyle.Render(f.value),
		))
	}

	return b.String()
}

func (m detailModel) renderVFConfig() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("  VF Configuration"))
	b.WriteString("\n\n")

	vfStatus := fmt.Sprintf("%d / %d", m.device.NumVFs, m.device.TotalVFs)
	b.WriteString(fmt.Sprintf("  %s %s\n",
		labelStyle.Render("Current VFs:"),
		valueStyle.Render(vfStatus),
	))

	b.WriteString(fmt.Sprintf("\n  %s %s\n",
		labelStyle.Render("Set VF Count:"),
		m.input.View(),
	))

	if m.message != "" {
		b.WriteString("\n")
		if m.isError {
			b.WriteString(errorStyle.Render("  " + m.message))
		} else {
			b.WriteString(successStyle.Render("  " + m.message))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m detailModel) renderVFTable() string {
	var b strings.Builder

	// Count bound/available
	bound := 0
	for _, vf := range m.vfs {
		if vf.PodName != "" {
			bound++
		}
	}

	b.WriteString(headerStyle.Render(fmt.Sprintf("  Virtual Functions (%d bound, %d available)",
		bound, len(m.vfs)-bound)))
	b.WriteString("\n\n")

	header := fmt.Sprintf("  %-4s %-16s %-11s %-19s %s",
		"VF", "BDF", "Driver", "MAC", "Status")
	b.WriteString(dimStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  " + strings.Repeat("─", 78)))
	b.WriteString("\n")

	maxVisible := 8
	start := m.vfScroll
	end := start + maxVisible
	if end > len(m.vfs) {
		end = len(m.vfs)
	}

	for _, vf := range m.vfs[start:end] {
		b.WriteString(m.renderVFRow(vf))
		b.WriteString("\n")
	}

	if len(m.vfs) > maxVisible {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ... showing %d-%d of %d VFs",
			start+1, end, len(m.vfs))))
		b.WriteString("\n")
	}

	return b.String()
}

func (m detailModel) renderVFRow(vf sriov.VFInfo) string {
	mac := vf.MAC
	if mac == "" {
		mac = "-"
	}

	var status string
	if vf.PodName != "" {
		podInfo := fmt.Sprintf("%s/%s", vf.PodNS, vf.PodName)
		if len(podInfo) > 30 {
			podInfo = podInfo[:30] + ".."
		}
		status = enabledStyle.Render("● ") + valueStyle.Render(podInfo)

		if vf.NetworkName != "" {
			qos := ""
			if vf.MaxTxRate > 0 {
				if vf.MinTxRate > 0 {
					qos = fmt.Sprintf(" [%dM-%dM]", vf.MinTxRate, vf.MaxTxRate)
				} else {
					qos = fmt.Sprintf(" [max %dM]", vf.MaxTxRate)
				}
			}
			status += dimStyle.Render(fmt.Sprintf(" via %s%s", vf.NetworkName, qos))
		}
	} else {
		status = dimStyle.Render("○ Available")
	}

	row := fmt.Sprintf("  vf%-2d %-16s %-11s %-19s %s",
		vf.Index, vf.BDF, vf.Driver, mac, status)

	return row
}
