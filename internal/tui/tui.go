package tui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ixianjia/ceph-calc/internal/calc"
)

type wizardInput struct {
	usable       string
	protection   calc.ProtectionMode
	replFactor   int
	ecK, ecM     int
	diskType     calc.DiskType
	diskSize     uint64
	disksPerNode int
	safetyRatio  float64
	nodeCount    int
	services     calc.AdditionalService
	release      calc.CephRelease
}

func prompt(scanner *bufio.Scanner, label, defaultValue string) string {
	val := defaultValue
	parts := strings.SplitN(label, "%s", 2)
	if len(parts) == 2 {
		label = parts[0] + val + parts[1]
	}
	fmt.Printf("  %-30s", strings.TrimSpace(label)+" : ")
	if scanner.Scan() {
		t := strings.TrimSpace(scanner.Text())
		if t != "" {
			val = t
		}
	}
	return val
}

func Run() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println()
	fmt.Println("  +-------------------------------------------+")
	fmt.Println("  |         Ceph Capacity Planner             |")
	fmt.Println("  +-------------------------------------------+")
	fmt.Println()

	in := wizardInput{
		disksPerNode: 12,
		safetyRatio:  0.80,
		diskSize:     16e12,
		diskType:     calc.DiskHDD,
		protection:   calc.ProtectionRepl,
		replFactor:   3,
	}

	// Step 1: Usable capacity
	fmt.Println("  Step 1/9: Target Usable Capacity")
	in.usable = prompt(scanner, "Capacity (e.g. 100TB, 1PB)", "100TB")
	if _, err := calc.ParseCapacity(in.usable); err != nil {
		fmt.Printf("  \n  ✖ Invalid: %v\n", err)
		return
	}
	fmt.Println()

	// Step 2: Protection strategy
	fmt.Println("  Step 2/9: Protection Strategy")
	fmt.Printf("    %s  %s\n", dim("1)"), "Replication x3  (3 copies, 66% overhead)")
	fmt.Printf("    %s  %s\n", dim("2)"), "Erasure Coding  (EC K+M, configurable)")
	protStr := prompt(scanner, "Choose [1-2]", "1")
	switch strings.TrimSpace(protStr) {
	case "2":
		in.protection = calc.ProtectionEC
		fmt.Println()
		fmt.Println("  Step 2b: EC Parameters (K/M)")
		ek := prompt(scanner, "Data chunks K", "4")
		em := prompt(scanner, "Parity chunks M", "2")
		in.ecK, _ = strconv.Atoi(ek)
		in.ecM, _ = strconv.Atoi(em)
		if in.ecK < 1 {
			in.ecK = 4
		}
		if in.ecM < 1 {
			in.ecM = 2
		}
	default:
		in.replFactor = 3
	}
	fmt.Println()

	// Step 3: Disk type
	fmt.Println("  Step 3/9: Disk Type")
	fmt.Printf("    %s  %s\n", dim("1)"), "HDD   (bulk storage, lowest cost)")
	fmt.Printf("    %s  %s\n", dim("2)"), "SSD   (balanced performance)")
	fmt.Printf("    %s  %s\n", dim("3)"), "NVMe  (high performance, DB/AI)")
	dtStr := prompt(scanner, "Choose [1-3]", "1")
	switch strings.TrimSpace(dtStr) {
	case "2":
		in.diskType = calc.DiskSSD
	case "3":
		in.diskType = calc.DiskNVME
	}
	fmt.Println()

	// Step 4: Disk size
	fmt.Println("  Step 4/9: Disk Size")
	sizes := []struct {
		label string
		bytes uint64
	}{
		{"  1TB", 1e12}, {"  2TB", 2e12}, {"  4TB", 4e12}, {"  8TB", 8e12},
		{" 12TB", 12e12}, {" 16TB", 16e12}, {" 18TB", 18e12},
		{" 20TB", 20e12}, {" 22TB", 22e12}, {" 24TB", 24e12},
	}
	fmt.Printf("    ")
	for i, s := range sizes {
		if i > 0 && i%5 == 0 {
			fmt.Printf("\n    ")
		}
		fmt.Printf("%s  %s  ", dim(fmt.Sprintf("%2d)", i+1)), s.label)
	}
	fmt.Println()
	defIdx := 5
	if in.diskType == calc.DiskNVME {
		defIdx = 2
	}
	szStr := prompt(scanner, "Choose", sizes[defIdx].label)
	szStr = strings.TrimSpace(szStr)
	szIdx, err := strconv.Atoi(szStr)
	if err != nil || szIdx < 1 || szIdx > len(sizes) {
		// try matching by label
		for i, s := range sizes {
			if strings.EqualFold(szStr, s.label) {
				szIdx = i + 1
				break
			}
		}
		if szIdx < 1 {
			szIdx = defIdx + 1
		}
	}
	in.diskSize = sizes[szIdx-1].bytes
	fmt.Println()

	// Step 5: Disks per node
	fmt.Println("  Step 5/9: Disks per Node")
	dpn := prompt(scanner, "Max disks per node [1-36]", "12")
	in.disksPerNode, _ = strconv.Atoi(dpn)
	if in.disksPerNode < 1 || in.disksPerNode > 36 {
		in.disksPerNode = 12
	}
	fmt.Println()

	// Step 6: Safety ratio
	fmt.Println("  Step 6/9: Safety Ratio")
	sr := prompt(scanner, "Safety ratio [0.50-0.95, default 0.80]", "0.80")
	in.safetyRatio, _ = strconv.ParseFloat(sr, 64)
	if in.safetyRatio < 0.50 || in.safetyRatio > 0.95 {
		in.safetyRatio = 0.80
	}
	fmt.Println()

	// Step 7: Additional services
	fmt.Println("  Step 7/9: Additional Services")
	fmt.Printf("    %s  %s\n", dim("1)"), "OSD only (baseline)")
	fmt.Printf("    %s  %s\n", dim("2)"), "+ CephFS  (MDS, file system metadata)")
	fmt.Printf("    %s  %s\n", dim("3)"), "+ RGW     (object gateway)")
	fmt.Printf("    %s  %s\n", dim("4)"), "+ CephFS + RGW")
	svcStr := prompt(scanner, "Choose [1-4]", "1")
	switch strings.TrimSpace(svcStr) {
	case "2":
		in.services = calc.ServiceCephFS
	case "3":
		in.services = calc.ServiceRGW
	case "4":
		in.services = calc.ServiceBoth
	}
	fmt.Println()

	// Step 8: Ceph release
	fmt.Println("  Step 8/9: Ceph Release")
	fmt.Printf("    %s  %s\n", dim("1)"), "Pacific (v16) — minor")
	fmt.Printf("    %s  %s\n", dim("2)"), "Quincy (v17) — stable")
	fmt.Printf("    %s  %s\n", dim("3)"), "Reef (v18, recommended)")
	fmt.Printf("    %s  %s\n", dim("4)"), "Squid (v19) — latest, Crimson OSD")
	rlsStr := prompt(scanner, "Choose [1-4]", "3")
	switch strings.TrimSpace(rlsStr) {
	case "1": in.release = calc.ReleasePacific
	case "2": in.release = calc.ReleaseQuincy
	case "3": in.release = calc.ReleaseReef
	case "4": in.release = calc.ReleaseSquid
	}
	fmt.Println()

	// Step 9: Node count
	fmt.Println("  Step 9/9: Node Count")
	nc := prompt(scanner, "Node count (0=auto)", "0")
	in.nodeCount, _ = strconv.Atoi(nc)
	if in.nodeCount < 0 {
		in.nodeCount = 0
	}
	fmt.Println()

	// Calculate after all inputs
	fmt.Println()
	fmt.Println("  Calculating...")
	fmt.Println()

	usableBytes, _ := calc.ParseCapacity(in.usable)
	calcResult := calc.Calculate(calc.Input{
		TargetUsable:       usableBytes,
		Protection:         in.protection,
		ReplFactor:         in.replFactor,
		ECK:                in.ecK,
		ECM:                in.ecM,
		DiskSize:           in.diskSize,
		DiskType:           in.diskType,
		DisksPerNode:       in.disksPerNode,
		SafetyRatio:        in.safetyRatio,
		NodeCount:          in.nodeCount,
		AdditionalServices: in.services,
		CephRelease:        in.release,
	})

	fmt.Println(printResult(in.usable, calcResult, sizes[szIdx-1].label))

	fmt.Println()
	fmt.Println("  ---- ctrl+c to exit ----")
	fmt.Scanln()
}

func dim(s string) string { return "\033[2m" + s + "\033[0m" }

func printResult(usable string, r calc.Result, diskLabel string) string {
	prot := ""
	if r.Input.Protection == calc.ProtectionRepl {
		prot = fmt.Sprintf("Replication x%d", r.Input.ReplFactor)
	} else {
		prot = fmt.Sprintf("EC %d+%d", r.Input.ECK, r.Input.ECM)
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf("  %s\n", bold("Target")))
	b.WriteString(fmt.Sprintf("    %-18s %s\n", "  Usable:", usable))
	b.WriteString(fmt.Sprintf("    %-18s %s\n", "  Protection:", prot))
	b.WriteString(fmt.Sprintf("    %-18s %s %s\n", "  Disk:", r.Input.DiskType, calc.FormatBytes(r.Input.DiskSize)))
	b.WriteString(fmt.Sprintf("    %-18s %.0f%%\n", "  Safety:", r.Input.SafetyRatio*100))
	releaseNames := map[calc.CephRelease]string{0: "Pacific", 1: "Quincy", 2: "Reef", 3: "Squid"}
	b.WriteString(fmt.Sprintf("    %-18s %s\n", "  Release:", releaseNames[r.Input.CephRelease]))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s\n", bold("Hardware Plan")))
	b.WriteString(fmt.Sprintf("    %-18s %d\n", "  Nodes:", r.Nodes))
	b.WriteString(fmt.Sprintf("    %-18s %d\n", "  OSDs per node:", r.OSDsPerNode))
	b.WriteString(fmt.Sprintf("    %-18s %d\n", "  Total OSDs:", r.TotalOSDs))
	b.WriteString(fmt.Sprintf("    %-18s %s\n", "  Raw:", calc.FormatBytes(r.RawCapacity)))
	b.WriteString(fmt.Sprintf("    %-18s %s\n", "  Usable:", calc.FormatBytes(r.UsableCapacity)))
	b.WriteString(fmt.Sprintf("    %-18s %.1f%%\n", "  Efficiency:", r.Efficiency))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s\n", bold("Node Spec (per node)")))
	b.WriteString(fmt.Sprintf("    %-18s %s\n", "  CPU:", fmt.Sprintf("%d cores (%s)", r.Hardware.CPUCoresPerNode, r.Hardware.CPUDesc)))
	b.WriteString(fmt.Sprintf("    %-18s %s\n", "  RAM:", fmt.Sprintf("%.0f GB (%s)", r.Hardware.RAMGBPerNode, r.Hardware.RAMDesc)))
	b.WriteString(fmt.Sprintf("    %-18s %s\n", "  OSDs:", fmt.Sprintf("%d × %s", r.OSDsPerNode, calc.FormatBytes(r.Input.DiskSize))))
	b.WriteString(fmt.Sprintf("    %-18s %s\n", "  Network:", fmt.Sprintf("%s public + %s cluster", r.Network.PublicSpeed, r.Network.ClusterSpeed)))
	b.WriteString(fmt.Sprintf("    %-18s %s\n", "  Services:", r.Hardware.ServicesDesc))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s\n", bold("DB/WAL")))
	b.WriteString(fmt.Sprintf("    %s\n", r.Hardware.DBWALDesc))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s\n", bold("Network Plan")))
	b.WriteString(fmt.Sprintf("    %-18s %s  %s\n", "  Public:", r.Network.PublicSpeed, r.Network.PublicNetwork))
	b.WriteString(fmt.Sprintf("    %-18s %s  %s\n", "  Cluster:", r.Network.ClusterSpeed, r.Network.ClusterNetwork))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s\n", bold("PG Suggestion")))
	for _, pg := range r.PGSuggestions {
		b.WriteString(fmt.Sprintf("    %-26s %d PGs\n", pg.PoolName, pg.PGCount))
	}
	for _, line := range strings.Split(r.Hardware.PGConfigDesc, "\n") {
		b.WriteString(fmt.Sprintf("  %s\n", dim("  " + line)))
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s\n", bold("Reusable CLI Command")))
	cmd := "ceph-calc"
	cmd += " --usable " + usable
	if r.Input.Protection == calc.ProtectionRepl {
		cmd += fmt.Sprintf(" --repl %d", r.Input.ReplFactor)
	} else {
		cmd += fmt.Sprintf(" --ec-k %d --ec-m %d", r.Input.ECK, r.Input.ECM)
	}
	cmd += fmt.Sprintf(" --disk %s --disk-type %s", strings.TrimSpace(diskLabel), r.Input.DiskType)
	cmd += fmt.Sprintf(" --disks-per-node %d", r.Input.DisksPerNode)
	b.WriteString(fmt.Sprintf("    %s\n", cmd))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s\n", bold("Generate Cephadm Spec")))
	b.WriteString(fmt.Sprintf("    %s --gen-spec > cluster.yaml\n", cmd))

	sep := strings.Repeat("-", 48)
	return sep + "\n" + b.String() + sep
}

func bold(s string) string { return "\033[1m" + s + "\033[0m" }
