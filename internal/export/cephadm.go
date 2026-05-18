package export

import (
	"fmt"
	"strings"

	"github.com/ixianjia/ceph-calc/internal/calc"
	"gopkg.in/yaml.v3"
)

func GenerateCephadmSpec(result calc.Result) string {
	var docs []string
	ip := func(i int) string { return fmt.Sprintf("10.0.1.%d", 10+i) }

	// 1. Global config
	docs = append(docs, mustYAML(map[string]any{
		"service_type": "config",
		"service_id":   "global",
		"config": map[string]any{
			"public_network":                   result.Network.PublicNetwork,
			"cluster_network":                  result.Network.ClusterNetwork,
			"osd_pool_default_pg_autoscale_mode": true,
			"osd_memory_target_autotune":        true,
		},
	}))

	// 2. Hosts
	for i := 0; i < result.Nodes; i++ {
		hostName := fmt.Sprintf("ceph-node-%02d", i)
		var labels []string
		if i < 3 {
			labels = append(labels, "mon")
			if i < 2 {
				labels = append(labels, "mgr")
			}
		}
		hasMDS := result.Input.AdditionalServices == calc.ServiceCephFS || result.Input.AdditionalServices == calc.ServiceBoth
		if hasMDS && i < 2 {
			labels = append(labels, "mds")
		}
		hasRGW := result.Input.AdditionalServices == calc.ServiceRGW || result.Input.AdditionalServices == calc.ServiceBoth
		if hasRGW && i < 3 {
			labels = append(labels, "rgw")
		}
		doc := map[string]any{
			"service_type": "host",
			"addr":         ip(i),
			"hostname":     hostName,
		}
		if len(labels) > 0 {
			doc["labels"] = labels
		}
		docs = append(docs, mustYAML(doc))
	}

	// 3. MON
	monDoc := map[string]any{
		"service_type": "mon",
		"placement":    map[string]any{"label": "mon", "count": 3},
	}
	docs = append(docs, mustYAML(monDoc))

	// 4. MGR
	mgrDoc := map[string]any{
		"service_type": "mgr",
		"placement":    map[string]any{"label": "mgr", "count": 2},
		"spec":         map[string]any{"allow_multiple_per_host": false},
	}
	docs = append(docs, mustYAML(mgrDoc))

	// 5. OSD spec
	osdPlacement := map[string]any{"host_pattern": "*"}
	var osdSpec map[string]any
	if result.Input.DiskType == calc.DiskHDD {
		osdSpec = map[string]any{
			"data_devices": map[string]any{"rotational": true},
			"osd_per_device": 1,
		}
		docs = append(docs, mustYAML(map[string]any{
			"service_type": "osd",
			"service_id":   "osd_spec_hdd",
			"placement":    osdPlacement,
			"spec":         osdSpec,
		}))
	} else if result.Input.DiskType == calc.DiskNVME {
		osdSpec = map[string]any{
			"data_devices":    map[string]any{"rotational": false},
			"osd_per_device": 1,
		}
		docs = append(docs, mustYAML(map[string]any{
			"service_type": "osd",
			"service_id":   "osd_spec_nvme",
			"placement":    osdPlacement,
			"spec":         osdSpec,
		}))
	} else {
		osdSpec = map[string]any{
			"data_devices":    map[string]any{"rotational": false},
			"osd_per_device": 1,
		}
		docs = append(docs, mustYAML(map[string]any{
			"service_type": "osd",
			"service_id":   "osd_spec_ssd",
			"placement":    osdPlacement,
			"spec":         osdSpec,
		}))
	}

	// 6. DB/WAL devices for HDD
	if result.Input.DiskType == calc.DiskHDD {
		dbwalSpec := map[string]any{
			"data_devices": map[string]any{"rotational": true},
			"db_devices":   map[string]any{"rotational": false, "limit": 4},
			"wal_devices":  map[string]any{"rotational": false, "limit": 8},
			"osd_per_device": 1,
		}
		docs = append(docs, mustYAML(map[string]any{
			"service_type": "osd",
			"service_id":   "osd_spec_hdd_dbwal",
			"placement":    osdPlacement,
			"spec":         dbwalSpec,
		}))
	}

	// 7. EC profile (comment only - profiles are set via CLI, not service spec)
	if result.Input.Protection == calc.ProtectionEC {
		ecProfile := result.Input
		comment := fmt.Sprintf(
			"# Oneliner:  ceph osd erasure-code-profile set default k=%d m=%d technique=reed_sol_van crush-failure-domain=host\n",
			ecProfile.ECK, ecProfile.ECM)
		comment += fmt.Sprintf(
			"# Profile:    k=%d m=%d technique=reed_sol_van\n",
			ecProfile.ECK, ecProfile.ECM)
		docs = append(docs, comment)
	}

	// 8. Pools
	for _, pg := range result.PGSuggestions {
		poolSpec := map[string]any{
			"pool":             pg.PoolName,
			"pg_autoscale_mode": true,
			"pg_num":           pg.PGCount,
		}
		if pg.IsEC {
			poolSpec["erasure_code_profile"] = "default"
		} else {
			replicaSize := result.Input.ReplFactor
			if replicaSize == 0 {
				replicaSize = 3
			}
			poolSpec["replica_size"] = replicaSize
		}
		docs = append(docs, mustYAML(map[string]any{
			"service_type": "pool",
			"service_id":   pg.PoolName,
			"spec":         poolSpec,
		}))
	}

	// 9. MDS (if CephFS)
	if result.Input.AdditionalServices == calc.ServiceCephFS || result.Input.AdditionalServices == calc.ServiceBoth {
		docs = append(docs, mustYAML(map[string]any{
			"service_type": "mds",
			"service_id":   "cephfs",
			"placement":    map[string]any{"label": "mds", "count": 2},
		}))
	}

	// 10. RGW (if RGW)
	if result.Input.AdditionalServices == calc.ServiceRGW || result.Input.AdditionalServices == calc.ServiceBoth {
		docs = append(docs, mustYAML(map[string]any{
			"service_type": "rgw",
			"service_id":   "rgw.default",
			"placement":    map[string]any{"label": "rgw", "count": 1},
			"spec": map[string]any{
				"rgw_frontend_port":        8080,
				"rgw_realm":                "default",
				"rgw_zone":                 "default",
				"rgw_zonegroup":            "default",
				"rgw_thread_pool_size":     512,
				"rgw_enable_usage_log":     true,
				"rgw_enable_ops_log":       true,
			},
		}))
	}

	// 11. Crash
	docs = append(docs, mustYAML(map[string]any{
		"service_type": "crash",
		"placement":    map[string]any{"host_pattern": "*"},
	}))

	// 12. Grafana / Monitoring
	docs = append(docs, mustYAML(map[string]any{
		"service_type": "grafana",
	}))

	docs = append(docs, mustYAML(map[string]any{
		"service_type": "prometheus",
	}))

	// Output bootstrap hint at top
	firstNodeIP := ip(0)
	header := fmt.Sprintf("# cephadm spec generated by ceph-calc\n")
	header += fmt.Sprintf("#\n")
	header += fmt.Sprintf("# Nodes:   %d × %s %s\n", result.Nodes, result.Input.DiskType, calc.FormatBytes(result.Input.DiskSize))
	header += fmt.Sprintf("# OSDs:    %d (%d per node)\n", result.TotalOSDs, result.OSDsPerNode)
	header += fmt.Sprintf("# Raw:     %s  →  Usable: %s (%.1f%%)\n",
		calc.FormatBytes(result.RawCapacity), calc.FormatBytes(result.UsableCapacity), result.Efficiency)
	header += fmt.Sprintf("# Network: Public=%s Cluster=%s\n", result.Network.PublicNetwork, result.Network.ClusterNetwork)
	header += fmt.Sprintf("# Release: %s\n",
		map[int]string{0: "Pacific", 1: "Quincy", 2: "Reef", 3: "Squid"}[int(result.Input.CephRelease)])
	header += fmt.Sprintf("#\n")
	header += fmt.Sprintf("# === Deployment Guide ===\n")
	header += fmt.Sprintf("#\n")
	header += fmt.Sprintf("#  0) Prerequisites:\n")
	header += fmt.Sprintf("#     - All nodes: CentOS 9 / Ubuntu 22.04+\n")
	header += fmt.Sprintf("#     - Docker or Podman installed\n")
	header += fmt.Sprintf("#     - SSH root access from bootstrap node to all others\n")
	header += fmt.Sprintf("#     - Hostnames: ceph-node-00 ~ ceph-node-%02d\n", result.Nodes-1)
	header += fmt.Sprintf("#     - Firewall: open ports 3300, 6789, 6800-7300, 8443, 9090, 3000\n")
	header += fmt.Sprintf("#\n")
	header += fmt.Sprintf("#  1) On bootstrap node, set up hostnames:\n")
	header += fmt.Sprintf("#     echo %s ceph-node-00  >> /etc/hosts\n", ip(0))
	for i := 1; i < result.Nodes && i < 3; i++ {
		header += fmt.Sprintf("#     echo %s ceph-node-%02d >> /etc/hosts\n", ip(i), i)
	}
	if result.Nodes > 3 {
		header += fmt.Sprintf("#     ... (see IPs for remaining %d nodes)\n", result.Nodes-3)
	}
	header += fmt.Sprintf("#\n")
	header += fmt.Sprintf("#  2) Bootstrap the first node:\n")
	header += fmt.Sprintf("#     cephadm bootstrap --mon-ip %s --initial-dashboard-user admin\n", firstNodeIP)
	header += fmt.Sprintf("#\n")
	header += fmt.Sprintf("#  3) Copy SSH key to remaining nodes:\n")
	header += fmt.Sprintf("#     ssh-copy-id -f -i /etc/ceph/ceph.pub root@ceph-node-01\n")
	header += fmt.Sprintf("#     ssh-copy-id -f -i /etc/ceph/ceph.pub root@ceph-node-02\n")
	if result.Nodes > 3 {
		header += fmt.Sprintf("#     ... (repeat for remaining nodes)\n")
	}
	header += fmt.Sprintf("#\n")
	header += fmt.Sprintf("#  4) Add hosts to cluster:\n")
	header += fmt.Sprintf("#     ceph orch host add ceph-node-00 %s --labels mon,mgr\n", ip(0))
	for i := 1; i < result.Nodes && i < 3; i++ {
		header += fmt.Sprintf("#     ceph orch host add ceph-node-%02d %s\n", i, ip(i))
	}
	if result.Nodes > 3 {
		header += fmt.Sprintf("#     ceph orch host add ceph-node-%02d <IP>\n", result.Nodes-1)
	}
	header += fmt.Sprintf("#\n")
	header += fmt.Sprintf("#  5) Deploy all services from this spec:\n")
	header += fmt.Sprintf("#     ceph orch apply -i cluster.yaml\n")
	header += fmt.Sprintf("#\n")
	header += fmt.Sprintf("#  6) Verify cluster health:\n")
	header += fmt.Sprintf("#     ceph -s\n")
	header += fmt.Sprintf("#     ceph osd tree\n")
	header += fmt.Sprintf("#     ceph orch ps\n")
	header += fmt.Sprintf("#\n")
	header += fmt.Sprintf("#  7) Access Dashboard:\n")
	header += fmt.Sprintf("#     https://%s:8443\n", firstNodeIP)
	header += fmt.Sprintf("#     User: admin / Password: <from bootstrap output>\n")

	return header + "\n" + strings.Join(docs, "\n---\n") + "\n"
}

func mustYAML(v any) string {
	b, err := yaml.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("yaml marshal: %v", err))
	}
	return strings.TrimRight(string(b), "\n")
}
