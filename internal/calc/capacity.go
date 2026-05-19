package calc

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

type ProtectionMode int

const (
	ProtectionRepl ProtectionMode = iota
	ProtectionEC
)

type DiskType string

const (
	DiskHDD  DiskType = "hdd"
	DiskSSD  DiskType = "ssd"
	DiskNVME DiskType = "nvme"
)

type AdditionalService int

const (
	ServiceOSDOnly  AdditionalService = 0
	ServiceCephFS   AdditionalService = 1
	ServiceRGW      AdditionalService = 2
	ServiceBoth     AdditionalService = 3
)

type CephRelease int

const (
	ReleasePacific CephRelease = 0
	ReleaseQuincy  CephRelease = 1
	ReleaseReef    CephRelease = 2
	ReleaseSquid   CephRelease = 3
)

type Input struct {
	TargetUsable       uint64
	Protection         ProtectionMode
	ReplFactor         int
	ECK                int
	ECM                int
	DiskSize           uint64
	DiskType           DiskType
	DisksPerNode       int
	SafetyRatio        float64
	NodeCount          int
	AdditionalServices AdditionalService
	CephRelease        CephRelease
}

type Result struct {
	Input          Input
	Nodes          int
	OSDsPerNode    int
	TotalOSDs      int
	RawCapacity    uint64
	UsableCapacity uint64
	Efficiency     float64
	Network        NetworkRecommendation
	PGSuggestions  []PGSuggestion
	Hardware       HardwareEstimate
	IsAutoNodes    bool
	MinSafeNodes   int
	NodeWarning    string
}

type HardwareEstimate struct {
	RAMGBPerNode    float64
	TotalRAMGB      float64
	CPUCoresPerNode int
	TotalCPUCores   int
	RAMDesc         string
	CPUDesc         string
	ServicesDesc    string
	DBWALDesc       string
	PGConfigDesc    string
}

type NetworkRecommendation struct {
	PublicNetwork  string
	ClusterNetwork string
	PublicSpeed    string
	ClusterSpeed   string
}

type PGSuggestion struct {
	PoolName string
	PGCount  int
	IsEC     bool
}

type CompareResult struct {
	Name   string
	Result Result
}

func ProtectionString(p ProtectionMode, repl int, ecK, ecM int) string {
	if p == ProtectionRepl {
		return fmt.Sprintf("Replication x%d", repl)
	}
	return fmt.Sprintf("EC %d+%d", ecK, ecM)
}

func ParseCapacity(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return 0, fmt.Errorf("empty capacity string")
	}

	numEnd := len(s)
	for i, c := range s {
		if !unicode.IsDigit(c) && c != '.' {
			numEnd = i
			break
		}
	}

	numStr := s[:numEnd]
	unitStr := strings.ToLower(strings.TrimSpace(s[numEnd:]))

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", numStr)
	}

	var multiplier uint64
	switch unitStr {
	case "tb", "t":
		multiplier = 1000 * 1000 * 1000 * 1000
	case "pb", "p":
		multiplier = 1000 * 1000 * 1000 * 1000 * 1000
	case "gb", "g":
		multiplier = 1000 * 1000 * 1000
	case "tib", "ti":
		multiplier = 1099511627776
	case "pib", "pi":
		multiplier = 1099511627776 * 1024
	case "gib", "gi":
		multiplier = 1073741824
	case "":
		multiplier = 1
	default:
		return 0, fmt.Errorf("unknown unit: %s (use TB, PB, GB, TiB, PiB)", unitStr)
	}

	return uint64(num * float64(multiplier)), nil
}

func FormatBytes(bytes uint64) string {
	tb := float64(bytes) / 1e12
	if tb >= 1 {
		return fmt.Sprintf("%.1f TB", tb)
	}
	gb := float64(bytes) / 1e9
	return fmt.Sprintf("%.1f GB", gb)
}

func Calculate(input Input) Result {
	var overheadFactor float64
	if input.Protection == ProtectionRepl {
		overheadFactor = float64(input.ReplFactor)
	} else {
		overheadFactor = float64(input.ECK+input.ECM) / float64(input.ECK)
	}

	rawNeeded := float64(input.TargetUsable) * overheadFactor / input.SafetyRatio
	rawNeededBytes := uint64(rawNeeded)

	totalOSDs := int(math.Ceil(float64(rawNeededBytes) / float64(input.DiskSize)))

	isAuto := input.NodeCount == 0
	nodes := input.NodeCount
	if isAuto {
		nodes = int(math.Ceil(float64(totalOSDs) / float64(input.DisksPerNode)))
		if nodes < 3 {
			nodes = 3
		}
	}

	minSafe := input.ReplFactor
	if input.Protection == ProtectionEC {
		minSafe = input.ECK + input.ECM
	}

	var nodeWarn string
	if !isAuto && nodes < minSafe {
		nodeWarn = fmt.Sprintf(
			"%s recommends ≥%d nodes for full fault isolation. %d nodes may reduce failure tolerance.",
			ProtectionString(input.Protection, input.ReplFactor, input.ECK, input.ECM),
			minSafe, nodes)
	}

	osdsPerNode := int(math.Ceil(float64(totalOSDs) / float64(nodes)))
	actualTotalOSDs := nodes * osdsPerNode
	actualRaw := uint64(actualTotalOSDs) * input.DiskSize

	var usable uint64
	if input.Protection == ProtectionRepl {
		usable = uint64(float64(actualRaw) / float64(input.ReplFactor) * input.SafetyRatio)
	} else {
		usable = uint64(float64(actualRaw) * float64(input.ECK) / float64(input.ECK+input.ECM) * input.SafetyRatio)
	}

	efficiency := float64(usable) / float64(actualRaw) * 100

	network := recommendNetwork(input, actualTotalOSDs)
	pgSuggestions := suggestPGs(actualTotalOSDs, input)
	hardware := estimateHardware(input, nodes, osdsPerNode, actualTotalOSDs)

	return Result{
		Input:          input,
		Nodes:          nodes,
		OSDsPerNode:    osdsPerNode,
		TotalOSDs:      actualTotalOSDs,
		RawCapacity:    actualRaw,
		UsableCapacity: usable,
		Efficiency:     efficiency,
		Network:        network,
		PGSuggestions:  pgSuggestions,
		Hardware:       hardware,
		IsAutoNodes:    isAuto,
		MinSafeNodes:   minSafe,
		NodeWarning:    nodeWarn,
	}
}

func CompareStrategies(input Input) []CompareResult {
	var results []CompareResult

	strategies := []struct {
		name       string
		protection ProtectionMode
		repl       int
		ecK        int
		ecM        int
	}{
		{"Repl x2", ProtectionRepl, 2, 0, 0},
		{"Repl x3", ProtectionRepl, 3, 0, 0},
		{"EC 2+1", ProtectionEC, 0, 2, 1},
		{"EC 4+2", ProtectionEC, 0, 4, 2},
		{"EC 6+2", ProtectionEC, 0, 6, 2},
		{"EC 8+3", ProtectionEC, 0, 8, 3},
	}

	for _, s := range strategies {
		cfg := input
		cfg.Protection = s.protection
		cfg.ReplFactor = s.repl
		cfg.ECK = s.ecK
		cfg.ECM = s.ecM
		results = append(results, CompareResult{
			Name:   s.name,
			Result: Calculate(cfg),
		})
	}

	return results
}

func GetPreset(name string) (Input, error) {
	presets := map[string]Input{
		"small": {
			TargetUsable: 50 * 1e12,
			Protection:   ProtectionRepl,
			ReplFactor:   3,
			DiskSize:     8 * 1e12,
			DiskType:     DiskHDD,
			DisksPerNode: 12,
			SafetyRatio:  0.80,
			NodeCount:    0,
		},
		"medium": {
			TargetUsable: 200 * 1e12,
			Protection:   ProtectionRepl,
			ReplFactor:   3,
			DiskSize:     16 * 1e12,
			DiskType:     DiskHDD,
			DisksPerNode: 12,
			SafetyRatio:  0.80,
			NodeCount:    0,
		},
		"large": {
			TargetUsable: 1000 * 1e12,
			Protection:   ProtectionEC,
			ECK:          4,
			ECM:          2,
			DiskSize:     16 * 1e12,
			DiskType:     DiskHDD,
			DisksPerNode: 12,
			SafetyRatio:  0.80,
			NodeCount:    0,
		},
		"all-nvme": {
			TargetUsable: 100 * 1e12,
			Protection:   ProtectionRepl,
			ReplFactor:   3,
			DiskSize:     4 * 1e12,
			DiskType:     DiskNVME,
			DisksPerNode: 8,
			SafetyRatio:  0.80,
			NodeCount:    0,
		},
	}

	p, ok := presets[name]
	if !ok {
		return Input{}, fmt.Errorf("unknown preset: %s (valid: small, medium, large, all-nvme)", name)
	}
	return p, nil
}
