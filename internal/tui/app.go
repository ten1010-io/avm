package tui

import (
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/joonseolee/avm/internal/sriov"
)

type Model struct {
	activeView viewType
	dashboard  dashboardModel
	detail     detailModel
	demoMode   bool
	width      int
	height     int
}

func NewModel(demoMode bool) Model {
	return Model{
		activeView: dashboardView,
		dashboard:  newDashboardModel(),
		demoMode:   demoMode,
	}
}

func (m Model) Init() tea.Cmd {
	return m.scanDevices()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case devicesScannedMsg:
		m.dashboard.iommu = msg.iommu
		m.dashboard.devices = msg.devices
		m.dashboard.err = msg.err
		m.dashboard.lastRefresh = time.Now()
		return m, nil

	case vfSetResultMsg:
		return m.handleVFResult(msg)

	case grubUpdateMsg:
		return m.handleGrubResult(msg)
	}

	if m.activeView == detailView {
		var cmd tea.Cmd
		m.detail.input, cmd = m.detail.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	switch m.activeView {
	case detailView:
		return m.detail.View()
	default:
		return m.dashboard.View()
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.activeView {
	case dashboardView:
		return m.handleDashboardKey(msg)
	case detailView:
		return m.handleDetailKey(msg)
	}
	return m, nil
}

func (m Model) handleDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Confirm dialog active
	if m.dashboard.confirmIOMMU {
		switch msg.String() {
		case "y":
			m.dashboard.confirmIOMMU = false
			if m.demoMode {
				m.dashboard.grubMessage = "✓ GRUB updated (demo). Reboot to apply."
				m.dashboard.grubIsError = false
				return m, nil
			}
			return m, func() tea.Msg {
				result := sriov.EnableIOMMUKernelParam()
				return grubUpdateMsg{result: result}
			}
		case "n", "esc":
			m.dashboard.confirmIOMMU = false
			return m, nil
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.dashboard.cursor > 0 {
			m.dashboard.cursor--
		}
		return m, nil

	case "down", "j":
		if m.dashboard.cursor < len(m.dashboard.devices)-1 {
			m.dashboard.cursor++
		}
		return m, nil

	case "enter":
		if len(m.dashboard.devices) > 0 {
			dev := m.dashboard.devices[m.dashboard.cursor]
			m.detail = newDetailModel(dev, m.demoMode)
			m.activeView = detailView
			return m, textinput.Blink
		}
		return m, nil

	case "e":
		if m.iommuCanEnable() {
			m.dashboard.confirmIOMMU = true
			m.dashboard.grubMessage = ""
		}
		return m, nil

	case "r":
		return m, m.scanDevices()
	}

	return m, nil
}

func (m Model) iommuCanEnable() bool {
	return m.dashboard.iommu.State == sriov.IOMMUPassthrough
}

func (m Model) handleGrubResult(msg grubUpdateMsg) (tea.Model, tea.Cmd) {
	if msg.result.Success {
		m.dashboard.grubMessage = fmt.Sprintf(
			"✓ GRUB updated. Backup: %s. Reboot to apply intel_iommu=on",
			msg.result.BackupPath,
		)
		m.dashboard.grubIsError = false
	} else {
		m.dashboard.grubMessage = fmt.Sprintf("Error: %s", msg.result.ErrorMsg)
		m.dashboard.grubIsError = true
	}
	return m, nil
}

func (m Model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.activeView = dashboardView
		return m, nil

	case "ctrl+c":
		return m, tea.Quit

	case "enter":
		return m.applyVFCount()
	}

	var cmd tea.Cmd
	m.detail.input, cmd = m.detail.input.Update(msg)
	return m, cmd
}

func (m Model) applyVFCount() (tea.Model, tea.Cmd) {
	val := m.detail.input.Value()
	count, err := strconv.Atoi(val)
	if err != nil {
		m.detail.message = fmt.Sprintf("Invalid number: %s", val)
		m.detail.isError = true
		return m, nil
	}

	if count < 0 || count > m.detail.device.TotalVFs {
		m.detail.message = fmt.Sprintf("Must be between 0 and %d", m.detail.device.TotalVFs)
		m.detail.isError = true
		return m, nil
	}

	if m.demoMode {
		m.detail.device.NumVFs = count
		m.detail.message = fmt.Sprintf("✓ VF count set to %d (demo)", count)
		m.detail.isError = false
		m.updateDeviceInList(m.detail.device)
		return m, nil
	}

	bdf := m.detail.device.BDF
	return m, func() tea.Msg {
		err := sriov.SetVFCount(bdf, count)
		return vfSetResultMsg{bdf: bdf, count: count, err: err}
	}
}

func (m *Model) updateDeviceInList(dev sriov.Device) {
	for i, d := range m.dashboard.devices {
		if d.BDF == dev.BDF {
			m.dashboard.devices[i] = dev
			break
		}
	}
}

func (m Model) handleVFResult(msg vfSetResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.detail.message = fmt.Sprintf("Error: %s", msg.err.Error())
		m.detail.isError = true
	} else {
		m.detail.device.NumVFs = msg.count
		m.detail.message = fmt.Sprintf("✓ VF count set to %d", msg.count)
		m.detail.isError = false
		m.updateDeviceInList(m.detail.device)
	}
	return m, nil
}

func (m Model) scanDevices() tea.Cmd {
	demoMode := m.demoMode
	return func() tea.Msg {
		if demoMode {
			return devicesScannedMsg{
				iommu:   sriov.DemoIOMMUStatus(),
				devices: sriov.DemoDevices(),
			}
		}

		iommu := sriov.DetectIOMMU()
		devices, err := sriov.ScanDevices()
		return devicesScannedMsg{
			iommu:   iommu,
			devices: devices,
			err:     err,
		}
	}
}
