package sriov

func DemoIOMMUStatus() IOMMUStatus {
	return IOMMUStatus{
		State:      IOMMUEnabled,
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
			NumVFs:    4,
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

func DemoVFs(pfBDF string) []VFInfo {
	switch pfBDF {
	case "0000:03:00.0":
		return []VFInfo{
			{Index: 0, BDF: "0000:03:02.0", Driver: "vfio-pci", MAC: "aa:bb:cc:dd:ee:01",
				PodName: "ml-worker-0", PodNS: "ml-training", NetworkName: "high-perf-net",
				MinTxRate: 1000, MaxTxRate: 5000, VLAN: 100, SpoofChk: "on", Trust: "off"},
			{Index: 1, BDF: "0000:03:02.1", Driver: "vfio-pci", MAC: "aa:bb:cc:dd:ee:02",
				PodName: "ml-worker-1", PodNS: "ml-training", NetworkName: "high-perf-net",
				MinTxRate: 1000, MaxTxRate: 5000, VLAN: 100, SpoofChk: "on", Trust: "off"},
			{Index: 2, BDF: "0000:03:02.2", Driver: "i40e", MAC: "00:00:00:00:00:00"},
			{Index: 3, BDF: "0000:03:02.3", Driver: "i40e", MAC: "00:00:00:00:00:00"},
		}
	case "0000:03:00.1":
		return []VFInfo{
			{Index: 0, BDF: "0000:03:0a.0", Driver: "vfio-pci", MAC: "aa:bb:cc:dd:ff:01",
				PodName: "log-collector-0", PodNS: "monitoring", NetworkName: "standard-net",
				MinTxRate: 0, MaxTxRate: 1000, VLAN: 200, SpoofChk: "on", Trust: "off"},
			{Index: 1, BDF: "0000:03:0a.1", Driver: "i40e", MAC: "00:00:00:00:00:00"},
		}
	case "0000:82:00.0":
		return []VFInfo{
			{Index: 0, BDF: "0000:82:00.4", Driver: "vfio-pci", MAC: "",
				PodName: "gpu-train-0", PodNS: "ml-training", NetworkName: "gpu-net"},
			{Index: 1, BDF: "0000:82:00.5", Driver: "vfio-pci", MAC: "",
				PodName: "gpu-train-1", PodNS: "ml-training", NetworkName: "gpu-net"},
			{Index: 2, BDF: "0000:82:00.6", Driver: "nvidia", MAC: ""},
			{Index: 3, BDF: "0000:82:00.7", Driver: "nvidia", MAC: ""},
		}
	case "0000:af:00.0":
		return []VFInfo{
			{Index: 0, BDF: "0000:af:00.1", Driver: "vfio-pci", MAC: "52:54:00:a1:b2:01",
				PodName: "dpdk-app-0", PodNS: "network", NetworkName: "dpdk-net",
				MinTxRate: 5000, MaxTxRate: 25000, VLAN: 300, SpoofChk: "off", Trust: "on"},
			{Index: 1, BDF: "0000:af:00.2", Driver: "vfio-pci", MAC: "52:54:00:a1:b2:02",
				PodName: "dpdk-app-1", PodNS: "network", NetworkName: "dpdk-net",
				MinTxRate: 5000, MaxTxRate: 25000, VLAN: 300, SpoofChk: "off", Trust: "on"},
			{Index: 2, BDF: "0000:af:00.3", Driver: "vfio-pci", MAC: "52:54:00:a1:b2:03",
				PodName: "web-proxy-0", PodNS: "ingress", NetworkName: "standard-net",
				MinTxRate: 0, MaxTxRate: 10000, VLAN: 200},
			{Index: 3, BDF: "0000:af:00.4", Driver: "mlx5_core", MAC: "00:00:00:00:00:00"},
			{Index: 4, BDF: "0000:af:00.5", Driver: "mlx5_core", MAC: "00:00:00:00:00:00"},
			{Index: 5, BDF: "0000:af:00.6", Driver: "mlx5_core", MAC: "00:00:00:00:00:00"},
			{Index: 6, BDF: "0000:af:00.7", Driver: "mlx5_core", MAC: "00:00:00:00:00:00"},
			{Index: 7, BDF: "0000:af:01.0", Driver: "mlx5_core", MAC: "00:00:00:00:00:00"},
		}
	}
	return nil
}
