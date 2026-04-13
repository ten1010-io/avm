package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/joonseolee/avm/internal/sriov"
)

type pkeyModel struct {
	smName     string
	smActive   bool
	confPath   string
	partitions []sriov.PKeyPartition
	cursor     int
	err        error
	demoMode   bool

	// Add form state
	showAddForm bool
	nameInput   textinput.Model
	pkeyInput   textinput.Model
	focusIndex  int // 0=name, 1=pkey
	formMessage string
	formIsError bool

	// Delete confirm
	confirmDelete bool
}

func newPKeyModel(demoMode bool) pkeyModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "partition name"
	nameInput.CharLimit = 32
	nameInput.Width = 25

	pkeyInput := textinput.New()
	pkeyInput.Placeholder = "0x8001"
	pkeyInput.CharLimit = 6
	pkeyInput.Width = 10

	m := pkeyModel{
		demoMode:  demoMode,
		nameInput: nameInput,
		pkeyInput: pkeyInput,
	}

	m.loadData()
	return m
}

func (m *pkeyModel) loadData() {
	if m.demoMode {
		m.smName, m.smActive = sriov.DemoSubnetManager()
		m.partitions = sriov.DemoPKeyPartitions()
		m.confPath = "/etc/opensm/partitions.conf"
		return
	}

	m.smName, m.smActive = sriov.DetectSubnetManager()
	partitions, confPath, err := sriov.ReadPKeyPartitions()
	m.partitions = partitions
	m.confPath = confPath
	m.err = err
}

func (m pkeyModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("P-Key Partitions"))
	b.WriteString("\n\n")

	// Subnet Manager status
	if m.smActive {
		b.WriteString(fmt.Sprintf("  %s %s\n",
			labelStyle.Render("Subnet Mgr:"),
			enabledStyle.Render(fmt.Sprintf("%s (active)", m.smName)),
		))
	} else {
		b.WriteString(fmt.Sprintf("  %s %s\n",
			labelStyle.Render("Subnet Mgr:"),
			disabledStyle.Render(fmt.Sprintf("%s (inactive)", m.smName)),
		))
	}

	if m.confPath != "" {
		b.WriteString(fmt.Sprintf("  %s %s\n",
			labelStyle.Render("Config:"),
			dimStyle.Render(m.confPath),
		))
	}
	b.WriteString("\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %s", m.err.Error())))
		b.WriteString("\n\n")
	}

	// Delete confirmation
	if m.confirmDelete && len(m.partitions) > 0 {
		p := m.partitions[m.cursor]
		b.WriteString(warningStyle.Render(fmt.Sprintf(
			"  ⚠ Delete P-Key %s (%s)? This will restart OpenSM.", p.PKey, p.Name)))
		b.WriteString("\n\n")
		b.WriteString(headerStyle.Render("  [y] Yes  [n] Cancel"))
		b.WriteString("\n")
		return boxStyle.Render(b.String())
	}

	// Add form
	if m.showAddForm {
		b.WriteString(m.renderAddForm())
		return boxStyle.Render(b.String())
	}

	// Partition table
	if len(m.partitions) == 0 {
		b.WriteString(dimStyle.Render("  No P-Key partitions found."))
		b.WriteString("\n")
	} else {
		b.WriteString(m.renderPartitionTable())
	}

	if m.formMessage != "" {
		if m.formIsError {
			b.WriteString(errorStyle.Render("  " + m.formMessage))
		} else {
			b.WriteString(successStyle.Render("  " + m.formMessage))
		}
		b.WriteString("\n")
	}

	if m.demoMode {
		b.WriteString("\n")
		b.WriteString(warningStyle.Render("  [Demo Mode] P-Key changes are simulated"))
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("  [a] Add P-Key  [d] Delete  [r] Refresh  [Esc] Back"))

	return boxStyle.Render(b.String())
}

func (m pkeyModel) renderPartitionTable() string {
	var b strings.Builder

	header := fmt.Sprintf("  %-10s %-20s %-20s %s",
		"P-Key", "Name", "Members", "Status")
	b.WriteString(dimStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  " + strings.Repeat("─", 60)))
	b.WriteString("\n")

	for i, p := range m.partitions {
		members := p.Members
		if len(members) > 18 {
			members = members[:18] + ".."
		}

		name := p.Name
		if name == "" {
			name = "-"
		}

		var status string
		if p.Active {
			status = enabledStyle.Render("● Active")
		} else {
			status = dimStyle.Render("○ Inactive")
		}

		row := fmt.Sprintf("  %-10s %-20s %-20s %s",
			p.PKey, name, members, status)

		if i == m.cursor {
			b.WriteString(selectedRowStyle.Render("▸ " + row[2:]))
		} else {
			b.WriteString(normalRowStyle.Render(row))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m pkeyModel) renderAddForm() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("  Add P-Key Partition"))
	b.WriteString("\n\n")

	nameLabel := labelStyle.Render("Name:")
	pkeyLabel := labelStyle.Render("P-Key:")

	if m.focusIndex == 0 {
		nameLabel = headerStyle.Render("Name:")
	} else {
		pkeyLabel = headerStyle.Render("P-Key:")
	}

	b.WriteString(fmt.Sprintf("  %s    %s\n", nameLabel, m.nameInput.View()))
	b.WriteString(fmt.Sprintf("  %s   %s\n", pkeyLabel, m.pkeyInput.View()))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Members: ALL=full (default)"))
	b.WriteString("\n")

	if m.formMessage != "" {
		b.WriteString("\n")
		if m.formIsError {
			b.WriteString(errorStyle.Render("  " + m.formMessage))
		} else {
			b.WriteString(successStyle.Render("  " + m.formMessage))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  [Tab] Next field  [Enter] Create  [Esc] Cancel"))

	return b.String()
}
