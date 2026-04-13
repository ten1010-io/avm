package tui

import "github.com/joonseolee/avm/internal/sriov"

type viewType int

const (
	dashboardView viewType = iota
	detailView
	pkeyView
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

type grubUpdateMsg struct {
	result sriov.GrubUpdateResult
}

type pkeyAddResultMsg struct {
	name string
	pkey string
	err  error
}

type pkeyDeleteResultMsg struct {
	pkey string
	err  error
}
