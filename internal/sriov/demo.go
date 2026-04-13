package sriov

func DemoIOMMUStatus() IOMMUStatus {
	return IOMMUStatus{
		Enabled:    true,
		Method:     "intel_iommu=on",
		HasGroups:  true,
		GroupCount: 24,
	}
}

func DemoDevices() []Device {
	return []Device{
		{
			BDF:       "0000:03:00.0",
			VendorID:  "8086",
			DeviceID:  "1572",
			Vendor:    "Intel Corporation",
			DevName:   "Ethernet Controller X710 for 10GbE SFP+",
			Driver:    "i40e",
			Class:     "Ethernet controller",
			TotalVFs:  64,
			NumVFs:    0,
			NetIfaces: []string{"ens3f0"},
		},
		{
			BDF:       "0000:03:00.1",
			VendorID:  "8086",
			DeviceID:  "1572",
			Vendor:    "Intel Corporation",
			DevName:   "Ethernet Controller X710 for 10GbE SFP+",
			Driver:    "i40e",
			Class:     "Ethernet controller",
			TotalVFs:  64,
			NumVFs:    2,
			NetIfaces: []string{"ens3f1"},
		},
		{
			BDF:       "0000:82:00.0",
			VendorID:  "10de",
			DeviceID:  "20b5",
			Vendor:    "NVIDIA Corporation",
			DevName:   "A100 PCIe 80GB",
			Driver:    "nvidia",
			Class:     "3D controller",
			TotalVFs:  16,
			NumVFs:    4,
			NetIfaces: nil,
		},
		{
			BDF:       "0000:af:00.0",
			VendorID:  "15b3",
			DeviceID:  "101b",
			Vendor:    "Mellanox Technologies",
			DevName:   "ConnectX-6 Dx EN 100G",
			Driver:    "mlx5_core",
			Class:     "Ethernet controller",
			TotalVFs:  128,
			NumVFs:    8,
			NetIfaces: []string{"enp175s0f0"},
		},
	}
}
