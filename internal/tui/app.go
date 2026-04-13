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
	pkey       pkeyModel
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

	case pkeyAddResultMsg:
		return m.handlePKeyAddResult(msg)

	case pkeyDeleteResultMsg:
		return m.handlePKeyDeleteResult(msg)
	}

	// Forward to active view's text inputs
	switch m.activeView {
	case detailView:
		var cmd tea.Cmd
		m.detail.input, cmd = m.detail.input.Update(msg)
		return m, cmd
	case pkeyView:
		if m.pkey.showAddForm {
			return m.updatePKeyFormInputs(msg)
		}
	}

	return m, nil
}

func (m Model) View() string {
	switch m.activeView {
	case detailView:
		return m.detail.View()
	case pkeyView:
		return m.pkey.View()
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
	case pkeyView:
		return m.handlePKeyKey(msg)
	}
	return m, nil
}

// ── Dashboard ──

func (m Model) handleDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	case "p":
		m.pkey = newPKeyModel(m.demoMode)
		m.activeView = pkeyView
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

// ── Device Detail ──

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

// ── P-Key ──

func (m Model) handlePKeyKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Delete confirmation
	if m.pkey.confirmDelete {
		switch msg.String() {
		case "y":
			m.pkey.confirmDelete = false
			p := m.pkey.partitions[m.pkey.cursor]
			if m.demoMode {
				m.pkey.partitions = append(m.pkey.partitions[:m.pkey.cursor], m.pkey.partitions[m.pkey.cursor+1:]...)
				if m.pkey.cursor >= len(m.pkey.partitions) && m.pkey.cursor > 0 {
					m.pkey.cursor--
				}
				m.pkey.formMessage = fmt.Sprintf("✓ P-Key %s deleted (demo)", p.PKey)
				m.pkey.formIsError = false
				return m, nil
			}
			pkey := p.PKey
			return m, func() tea.Msg {
				err := sriov.DeletePKeyPartition(pkey)
				return pkeyDeleteResultMsg{pkey: pkey, err: err}
			}
		case "n", "esc":
			m.pkey.confirmDelete = false
			return m, nil
		}
		return m, nil
	}

	// Add form
	if m.pkey.showAddForm {
		return m.handlePKeyFormKey(msg)
	}

	// Normal list view
	switch msg.String() {
	case "esc":
		m.activeView = dashboardView
		return m, nil

	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.pkey.cursor > 0 {
			m.pkey.cursor--
		}
		return m, nil

	case "down", "j":
		if m.pkey.cursor < len(m.pkey.partitions)-1 {
			m.pkey.cursor++
		}
		return m, nil

	case "a":
		m.pkey.showAddForm = true
		m.pkey.nameInput.SetValue("")
		m.pkey.pkeyInput.SetValue("")
		m.pkey.focusIndex = 0
		m.pkey.nameInput.Focus()
		m.pkey.pkeyInput.Blur()
		m.pkey.formMessage = ""
		return m, textinput.Blink

	case "d":
		if len(m.pkey.partitions) > 0 {
			m.pkey.confirmDelete = true
		}
		return m, nil

	case "r":
		m.pkey.loadData()
		return m, nil
	}

	return m, nil
}

func (m Model) handlePKeyFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.pkey.showAddForm = false
		m.pkey.formMessage = ""
		return m, nil

	case "tab", "shift+tab":
		m.pkey.focusIndex = (m.pkey.focusIndex + 1) % 2
		if m.pkey.focusIndex == 0 {
			m.pkey.nameInput.Focus()
			m.pkey.pkeyInput.Blur()
		} else {
			m.pkey.nameInput.Blur()
			m.pkey.pkeyInput.Focus()
		}
		return m, textinput.Blink

	case "enter":
		return m.submitPKeyForm()
	}

	return m.updatePKeyFormInputs(msg)
}

func (m Model) updatePKeyFormInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.pkey.nameInput, cmd = m.pkey.nameInput.Update(msg)
	cmds = append(cmds, cmd)

	m.pkey.pkeyInput, cmd = m.pkey.pkeyInput.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) submitPKeyForm() (tea.Model, tea.Cmd) {
	name := m.pkey.nameInput.Value()
	pkey := m.pkey.pkeyInput.Value()

	if name == "" {
		m.pkey.formMessage = "Name is required"
		m.pkey.formIsError = true
		return m, nil
	}

	if pkey == "" {
		m.pkey.formMessage = "P-Key value is required"
		m.pkey.formIsError = true
		return m, nil
	}

	if m.demoMode {
		newPartition := sriov.PKeyPartition{
			PKey:    pkey,
			Name:    name,
			Members: "ALL=full",
			Active:  true,
		}
		m.pkey.partitions = append(m.pkey.partitions, newPartition)
		m.pkey.showAddForm = false
		m.pkey.formMessage = fmt.Sprintf("✓ P-Key %s (%s) created (demo)", pkey, name)
		m.pkey.formIsError = false
		return m, nil
	}

	return m, func() tea.Msg {
		err := sriov.AddPKeyPartition(name, pkey, "ALL=full")
		return pkeyAddResultMsg{name: name, pkey: pkey, err: err}
	}
}

func (m Model) handlePKeyAddResult(msg pkeyAddResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.pkey.formMessage = fmt.Sprintf("Error: %s", msg.err.Error())
		m.pkey.formIsError = true
	} else {
		m.pkey.showAddForm = false
		m.pkey.formMessage = fmt.Sprintf("✓ P-Key %s (%s) created. OpenSM restarted.", msg.pkey, msg.name)
		m.pkey.formIsError = false
		m.pkey.loadData() // Reload
	}
	return m, nil
}

func (m Model) handlePKeyDeleteResult(msg pkeyDeleteResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.pkey.formMessage = fmt.Sprintf("Error: %s", msg.err.Error())
		m.pkey.formIsError = true
	} else {
		m.pkey.formMessage = fmt.Sprintf("✓ P-Key %s deleted. OpenSM restarted.", msg.pkey)
		m.pkey.formIsError = false
		m.pkey.loadData() // Reload
	}
	return m, nil
}

// ── Scan ──

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
