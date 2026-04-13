package sriov

import (
	"encoding/json"
	"os/exec"
	"strings"
)

type SriovNetworkInfo struct {
	Name      string
	Resource  string // resourceName from Policy
	VLAN      int
	MinTxRate int
	MaxTxRate int
	SpoofChk  string
	Trust     string
}

type PodVFBinding struct {
	PodName     string
	PodNS       string
	NetworkName string
	VFBDF       string
}

func FetchSriovNetworks() ([]SriovNetworkInfo, error) {
	out, err := exec.Command("kubectl", "get", "sriovnetwork",
		"--all-namespaces", "-o", "json").Output()
	if err != nil {
		return nil, err
	}

	var result struct {
		Items []struct {
			Metadata struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
			} `json:"metadata"`
			Spec struct {
				ResourceName string `json:"resourceName"`
				Vlan         int    `json:"vlan"`
				MinTxRate    int    `json:"minTxRate"`
				MaxTxRate    int    `json:"maxTxRate"`
				SpoofChk     string `json:"spoofChk"`
				Trust        string `json:"trust"`
			} `json:"spec"`
		} `json:"items"`
	}

	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	networks := make([]SriovNetworkInfo, 0, len(result.Items))
	for _, item := range result.Items {
		networks = append(networks, SriovNetworkInfo{
			Name:      item.Metadata.Name,
			Resource:  item.Spec.ResourceName,
			VLAN:      item.Spec.Vlan,
			MinTxRate: item.Spec.MinTxRate,
			MaxTxRate: item.Spec.MaxTxRate,
			SpoofChk:  item.Spec.SpoofChk,
			Trust:     item.Spec.Trust,
		})
	}

	return networks, nil
}

func FetchPodVFBindings() ([]PodVFBinding, error) {
	out, err := exec.Command("kubectl", "get", "pods",
		"--all-namespaces", "-o", "json").Output()
	if err != nil {
		return nil, err
	}

	var result struct {
		Items []struct {
			Metadata struct {
				Name        string            `json:"name"`
				Namespace   string            `json:"namespace"`
				Annotations map[string]string `json:"annotations"`
			} `json:"metadata"`
		} `json:"items"`
	}

	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	bindings := make([]PodVFBinding, 0)
	for _, pod := range result.Items {
		networkAnnotation := pod.Metadata.Annotations["k8s.v1.cni.cncf.io/network-status"]
		if networkAnnotation == "" {
			continue
		}

		var netStatuses []struct {
			Name      string `json:"name"`
			DeviceInfo struct {
				Type    string `json:"type"`
				Version string `json:"version"`
				PCI     struct {
					PCIBDF string `json:"pci-address"`
				} `json:"pci"`
			} `json:"device-info"`
		}

		if err := json.Unmarshal([]byte(networkAnnotation), &netStatuses); err != nil {
			continue
		}

		for _, ns := range netStatuses {
			pciAddr := ns.DeviceInfo.PCI.PCIBDF
			if pciAddr == "" {
				continue
			}

			bindings = append(bindings, PodVFBinding{
				PodName:     pod.Metadata.Name,
				PodNS:       pod.Metadata.Namespace,
				NetworkName: extractNetworkName(ns.Name),
				VFBDF:       pciAddr,
			})
		}
	}

	return bindings, nil
}

func EnrichVFsWithK8sInfo(vfs []VFInfo, bindings []PodVFBinding, networks []SriovNetworkInfo) []VFInfo {
	enriched := make([]VFInfo, len(vfs))
	copy(enriched, vfs)

	networkMap := make(map[string]SriovNetworkInfo, len(networks))
	for _, n := range networks {
		networkMap[n.Name] = n
	}

	for i := range enriched {
		for _, b := range bindings {
			if b.VFBDF == enriched[i].BDF {
				enriched[i].PodName = b.PodName
				enriched[i].PodNS = b.PodNS
				enriched[i].NetworkName = b.NetworkName

				if net, ok := networkMap[b.NetworkName]; ok {
					enriched[i].MinTxRate = net.MinTxRate
					enriched[i].MaxTxRate = net.MaxTxRate
					enriched[i].VLAN = net.VLAN
					enriched[i].SpoofChk = net.SpoofChk
					enriched[i].Trust = net.Trust
				}
				break
			}
		}
	}

	return enriched
}

func extractNetworkName(fullName string) string {
	parts := strings.Split(fullName, "/")
	if len(parts) == 2 {
		return parts[1]
	}
	return fullName
}
