package tui

import "github.com/joonseolee/avm/internal/sriov"

type viewType int

const (
	dashboardView viewType = iota
	detailView
)

type devicesScannedMsg struct {
	iommu   sriov.IOMMUStatus
	devices []sriov.Device
	err     error
}

type vfSetResultMsg struct {
	bdf   string
	count int
	err   error
}

type navigateMsg struct {
	view        viewType
	deviceIndex int
}
