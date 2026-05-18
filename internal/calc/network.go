package calc

func recommendNetwork(input Input, totalOSDs int) NetworkRecommendation {
	if input.DiskType == DiskNVME {
		if totalOSDs > 8 {
			return NetworkRecommendation{
				PublicNetwork:  "10.0.1.0/24",
				ClusterNetwork: "10.0.2.0/24",
				PublicSpeed:    "25GbE",
				ClusterSpeed:   "100GbE (RDMA recommended per Ceph docs)",
			}
		}
		return NetworkRecommendation{
			PublicNetwork:  "10.0.1.0/24",
			ClusterNetwork: "10.0.2.0/24",
			PublicSpeed:    "25GbE",
			ClusterSpeed:   "25GbE",
		}
	}

	if input.DiskType == DiskSSD {
		return NetworkRecommendation{
			PublicNetwork:  "10.0.1.0/24",
			ClusterNetwork: "10.0.2.0/24",
			PublicSpeed:    "25GbE",
			ClusterSpeed:   "25GbE",
		}
	}

	switch {
	case totalOSDs > 50:
		return NetworkRecommendation{
			PublicNetwork:  "10.0.1.0/24",
			ClusterNetwork: "10.0.2.0/24",
			PublicSpeed:    "10GbE",
			ClusterSpeed:   "25GbE",
		}
	case totalOSDs > 20:
		return NetworkRecommendation{
			PublicNetwork:  "10.0.1.0/24",
			ClusterNetwork: "10.0.2.0/24",
			PublicSpeed:    "10GbE",
			ClusterSpeed:   "10GbE",
		}
	default:
		return NetworkRecommendation{
			PublicNetwork:  "10.0.1.0/24",
			ClusterNetwork: "10.0.2.0/24",
			PublicSpeed:    "10GbE",
			ClusterSpeed:   "10GbE",
		}
	}
}
