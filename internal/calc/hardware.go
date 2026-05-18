package calc

import "fmt"

func estimateHardware(input Input, nodes, osdsPerNode, totalOSDs int) HardwareEstimate {
	var ramPerOSD, cpuThreadsPerOSD float64
	var cpuDesc, dbwalDesc string

	switch input.DiskType {
	case DiskNVME:
		ramPerOSD = 6
		cpuThreadsPerOSD = 6
		cpuDesc = fmt.Sprintf("NVMe: %d threads (%.0f cores) per OSD", int(cpuThreadsPerOSD), cpuThreadsPerOSD/2)
		if input.CephRelease >= ReleaseSquid {
			dbwalDesc = "NVMe: consider Crimson OSD. No DB/WAL offload needed."
		} else {
			dbwalDesc = "NVMe OSDs do not need DB/WAL offload."
		}
	case DiskSSD:
		ramPerOSD = 4
		cpuThreadsPerOSD = 4
		cpuDesc = fmt.Sprintf("SSD: %d threads (%.0f cores) per OSD", int(cpuThreadsPerOSD), cpuThreadsPerOSD/2)
		dbwalDesc = "SSD OSDs generally do not need DB/WAL offload."
	default:
		ramPerOSD = 4
		cpuThreadsPerOSD = 3
		cpuDesc = fmt.Sprintf("HDD: %d threads (%.0f cores) per OSD", int(cpuThreadsPerOSD), cpuThreadsPerOSD/2)
		dbwalDesc = "DB/WAL offload recommended: 1x SSD per HDD OSD, or ~5 HDD OSDs per SATA SSD, up to 15 per NVMe SSD"
	}

	memoryFactor := 2.0
	if input.DiskType == DiskNVME {
		memoryFactor = 2.5
	}

	ramNeeded := float64(osdsPerNode)*ramPerOSD*memoryFactor + 4.0
	ramNeeded *= 1.2

	threadsNeeded := float64(osdsPerNode)*cpuThreadsPerOSD + 3.0
	cpuCores := int(threadsNeeded/2 + 0.5)

	if cpuCores < 8 {
		cpuCores = 8
	}

	servicesDesc := "OSD only"
	if input.AdditionalServices == ServiceCephFS || input.AdditionalServices == ServiceBoth {
		ramNeeded += 8
		cpuCores += 4
		servicesDesc = "+ CephFS (MDS)"
	}
	if input.AdditionalServices == ServiceRGW || input.AdditionalServices == ServiceBoth {
		ramNeeded += 4
		cpuCores += 2
		if servicesDesc == "+ CephFS (MDS)" {
			servicesDesc = "+ CephFS + RGW"
		} else {
			servicesDesc = "+ RGW"
		}
	}

	totalRAM := ramNeeded * float64(nodes)
	totalCPU := cpuCores * nodes

	ramDesc := ""
	switch {
	case ramNeeded < 48:
		ramDesc = "64 GB"
	case ramNeeded < 80:
		ramDesc = "96 GB"
	case ramNeeded < 120:
		ramDesc = "128 GB"
	case ramNeeded < 160:
		ramDesc = fmt.Sprintf("192 GB (~%.0f GB calced)", ramNeeded)
	default:
		ramDesc = fmt.Sprintf("256 GB (~%.0f GB calced)", ramNeeded)
	}

	pgDesc := ""
	if input.CephRelease >= ReleaseReef {
		pgDesc = "Recommended: pg_autoscale_mode on (default since Reef).\n"
	} else {
		pgDesc = "PG autoscale mode available (enable manually: ceph config set global osd_pool_default_pg_autoscale_mode on).\n"
	}
	pgDesc += "  Manual: target ~100-200 PGs per OSD. Round to power of 2.\n"
	pgDesc += fmt.Sprintf("  For %d OSDs, total cluster PGs ≈ %d-%d.", totalOSDs, totalOSDs*100, totalOSDs*200)

	return HardwareEstimate{
		RAMGBPerNode:    ramNeeded,
		TotalRAMGB:      totalRAM,
		CPUCoresPerNode: cpuCores,
		TotalCPUCores:   totalCPU,
		RAMDesc:         ramDesc,
		CPUDesc:         cpuDesc,
		ServicesDesc:    servicesDesc,
		DBWALDesc:       dbwalDesc,
		PGConfigDesc:    pgDesc,
	}
}
