package cmd

import (
	"fmt"
	"os"

	"github.com/ixianjia/ceph-calc/internal/calc"
	"github.com/ixianjia/ceph-calc/internal/export"
	"github.com/ixianjia/ceph-calc/internal/tui"
	"github.com/spf13/cobra"
)

var (
	targetUsable       string
	replFactor         int
	ecK                int
	ecM                int
	diskSize           string
	diskType           string
	disksPerNode       int
	safetyRatio        float64
	nodeCount          int
	outputFormat       string
	compareMode        bool
	genSpec            bool
	preset             string
	additionalServices int
	cephRelease        int
)

var rootCmd = &cobra.Command{
	Use:   "ceph-calc",
	Short: "Ceph Capacity Planner - Interactive CLI tool for Ceph cluster sizing",
	RunE: func(cmd *cobra.Command, args []string) error {
		if compareMode {
			return runCompare()
		}
		if cmd.Flags().Changed("usable") || cmd.Flags().Changed("preset") {
			return runNonInteractive()
		}
		return runInteractive()
	},
}

func init() {
	rootCmd.Flags().StringVarP(&targetUsable, "usable", "u", "", "Target usable capacity (e.g. 100TB, 1PB)")
	rootCmd.Flags().IntVarP(&replFactor, "repl", "r", 0, "Replication factor (2 or 3)")
	rootCmd.Flags().IntVar(&ecK, "ec-k", 0, "Erasure coding K (data chunks)")
	rootCmd.Flags().IntVar(&ecM, "ec-m", 0, "Erasure coding M (parity chunks)")
	rootCmd.Flags().StringVar(&diskSize, "disk", "16TB", "Disk size (e.g. 4TB, 8TB, 16TB)")
	rootCmd.Flags().StringVar(&diskType, "disk-type", "hdd", "Disk type: hdd, ssd, nvme")
	rootCmd.Flags().IntVar(&disksPerNode, "disks-per-node", 12, "Max disks per node")
	rootCmd.Flags().Float64Var(&safetyRatio, "safety", 0.80, "Safety ratio (0.70-0.95)")
	rootCmd.Flags().IntVarP(&nodeCount, "nodes", "n", 0, "Fixed node count (0=auto)")
	rootCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table, json, yaml")
	rootCmd.Flags().BoolVar(&compareMode, "compare", false, "Compare all protection strategies")
	rootCmd.Flags().BoolVar(&genSpec, "gen-spec", false, "Generate cephadm spec file")
	rootCmd.Flags().StringVar(&preset, "preset", "", "Use preset: small, medium, large, all-nvme")
	rootCmd.Flags().IntVar(&additionalServices, "services", 0, "Additional services: 0=OSD only, 1=+CephFS, 2=+RGW, 3=+CephFS+RGW")
	rootCmd.Flags().IntVar(&cephRelease, "release", 2, "Ceph release: 0=Pacific, 1=Quincy, 2=Reef, 3=Squid")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runInteractive() error {
	tui.Run()
	return nil
}

func runNonInteractive() error {
	input, err := parseInput()
	if err != nil {
		return err
	}

	result := calc.Calculate(input)

	if genSpec {
		export.PrintCephadmSpec(result, os.Stdout)
		return nil
	}

	switch outputFormat {
	case "json":
		return export.ToJSON(result, os.Stdout)
	case "yaml":
		return export.ToYAML(result, os.Stdout)
	default:
		export.PrintTable(result, os.Stdout)
		return nil
	}
}

func runCompare() error {
	input, err := parseInput()
	if err != nil {
		return err
	}

	results := calc.CompareStrategies(input)
	export.PrintComparison(results, os.Stdout)
	return nil
}

func parseInput() (calc.Input, error) {
	if preset != "" {
		return calc.GetPreset(preset)
	}

	if targetUsable == "" {
		return calc.Input{}, fmt.Errorf("--usable is required in non-interactive mode")
	}

	usableBytes, err := calc.ParseCapacity(targetUsable)
	if err != nil {
		return calc.Input{}, fmt.Errorf("invalid capacity: %w", err)
	}

	diskBytes, err := calc.ParseCapacity(diskSize)
	if err != nil {
		return calc.Input{}, fmt.Errorf("invalid disk size: %w", err)
	}

		input := calc.Input{
			TargetUsable:       usableBytes,
			DiskSize:           diskBytes,
			DiskType:           calc.DiskType(diskType),
			DisksPerNode:       disksPerNode,
			SafetyRatio:        safetyRatio,
			NodeCount:          nodeCount,
			AdditionalServices: calc.AdditionalService(additionalServices),
			CephRelease:        calc.CephRelease(cephRelease),
		}

	if replFactor > 0 {
		input.Protection = calc.ProtectionRepl
		input.ReplFactor = replFactor
	} else if ecK > 0 && ecM > 0 {
		input.Protection = calc.ProtectionEC
		input.ECK = ecK
		input.ECM = ecM
	} else {
		input.Protection = calc.ProtectionRepl
		input.ReplFactor = 3
	}

	return input, nil
}
