package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/joonseolee/avm/internal/sriov"
)

type detailModel struct {
	device    sriov.Device
	input     textinput.Model
	message   string
	isError   bool
	demoMode  bool
}

func newDetailModel(dev sriov.Device, demoMode bool) detailModel {
	ti := textinput.New()
	ti.Placeholder = fmt.Sprintf("0-%d", dev.TotalVFs)
	ti.Focus()
	ti.CharLimit = 4
	ti.Width = 10
	ti.SetValue(fmt.Sprintf("%d", dev.NumVFs))

	return detailModel{
		device:   dev,
		input:    ti,
		demoMode: demoMode,
	}
}

func (m detailModel) View() string {
	var b strings.Builder

	title := fmt.Sprintf("Device: %s - %s", m.device.BDF, m.device.DevName)
	if len(title) > 55 {
		title = title[:55] + ".."
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

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

	b.WriteString("\n")
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

	if m.demoMode {
		b.WriteString("\n")
		b.WriteString(warningStyle.Render("  [Demo Mode] VF changes are simulated"))
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("  [Enter] Apply  [Esc] Back"))

	return boxStyle.Render(b.String())
}
