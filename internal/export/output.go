package export

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ixianjia/ceph-calc/internal/calc"
	"gopkg.in/yaml.v3"
)

func ToJSON(result calc.Result, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func ToYAML(result calc.Result, w io.Writer) error {
	enc := yaml.NewEncoder(w)
	defer enc.Close()
	return enc.Encode(result)
}

func PrintTable(result calc.Result, w io.Writer) {
	if w == nil {
		w = os.Stdout
	}

	protection := ""
	if result.Input.Protection == calc.ProtectionRepl {
		protection = fmt.Sprintf("Replication x%d", result.Input.ReplFactor)
	} else {
		protection = fmt.Sprintf("EC %d+%d", result.Input.ECK, result.Input.ECM)
	}

	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "╔══════════════════════════════════════════╗\n")
	fmt.Fprintf(w, "║       Ceph Capacity Plan Result          ║\n")
	fmt.Fprintf(w, "╠══════════════════════════════════════════╣\n")
	fmt.Fprintf(w, "║ Target Usable:    %-22s ║\n", calc.FormatBytes(result.Input.TargetUsable))
	fmt.Fprintf(w, "║ Protection:       %-22s ║\n", protection)
	fmt.Fprintf(w, "║ Disk Type:        %-22s ║\n", fmt.Sprintf("%s %s", result.Input.DiskType, calc.FormatBytes(result.Input.DiskSize)))
	fmt.Fprintf(w, "║ Safety Ratio:     %-22.0f%% ║\n", result.Input.SafetyRatio*100)
	fmt.Fprintf(w, "╠══════════════════════════════════════════╣\n")
	fmt.Fprintf(w, "║          Recommended Configuration       ║\n")
	fmt.Fprintf(w, "╠══════════════════════════════════════════╣\n")
	fmt.Fprintf(w, "║ Nodes:            %-22d ║\n", result.Nodes)
	fmt.Fprintf(w, "║ OSDs per node:    %-22d ║\n", result.OSDsPerNode)
	fmt.Fprintf(w, "║ Total OSDs:       %-22d ║\n", result.TotalOSDs)
	fmt.Fprintf(w, "║ Raw Capacity:     %-22s ║\n", calc.FormatBytes(result.RawCapacity))
	fmt.Fprintf(w, "║ Usable Capacity:  %-22s ║\n", calc.FormatBytes(result.UsableCapacity))
	fmt.Fprintf(w, "║ Efficiency:       %-21.1f%% ║\n", result.Efficiency)
	fmt.Fprintf(w, "╠══════════════════════════════════════════╣\n")
	fmt.Fprintf(w, "║           Node Spec (per node)           ║\n")
	fmt.Fprintf(w, "╠══════════════════════════════════════════╣\n")
	fmt.Fprintf(w, "║ CPU:              %-22s ║\n", fmt.Sprintf("%d cores", result.Hardware.CPUCoresPerNode))
	fmt.Fprintf(w, "║ Services:         %-22s ║\n", result.Hardware.ServicesDesc)
	fmt.Fprintf(w, "╠══════════════════════════════════════════╣\n")
	fmt.Fprintf(w, "║           DB/WAL Recommendation          ║\n")
	fmt.Fprintf(w, "╠══════════════════════════════════════════╣\n")
	fmt.Fprintf(w, "║ %-40s ║\n", result.Hardware.DBWALDesc)
	fmt.Fprintf(w, "╠══════════════════════════════════════════╣\n")
	fmt.Fprintf(w, "║           Network Recommendation         ║\n")
	fmt.Fprintf(w, "╠══════════════════════════════════════════╣\n")
	fmt.Fprintf(w, "║ Public:  %-10s  %-25s ║\n", result.Network.PublicSpeed, result.Network.PublicNetwork)
	fmt.Fprintf(w, "║ Cluster: %-10s  %-25s ║\n", result.Network.ClusterSpeed, result.Network.ClusterNetwork)
	fmt.Fprintf(w, "╠══════════════════════════════════════════╣\n")
	fmt.Fprintf(w, "║            PG Suggestions                ║\n")
	fmt.Fprintf(w, "╠══════════════════════════════════════════╣\n")
	for _, pg := range result.PGSuggestions {
		fmt.Fprintf(w, "║ %-20s %d PGs                    ║\n", pg.PoolName, pg.PGCount)
	}
	fmt.Fprintf(w, "╚══════════════════════════════════════════╝\n")
	fmt.Fprintf(w, "\n")
}

func PrintComparison(results []calc.CompareResult, w io.Writer) {
	if w == nil {
		w = os.Stdout
	}

	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, " ╔═══════════════════════════════════════════════════════════════════════════════════════╗\n")
	fmt.Fprintf(w, " ║                          Ceph Strategy Comparison                                    ║\n")
	fmt.Fprintf(w, " ╠═══════════╦═══════════╦═══════════╦═══════════╦═══════════╦══════════════╦═══════════╣\n")
	fmt.Fprintf(w, " ║ Strategy  ║  Nodes    ║ OSDs/Node ║ Total OSD ║ Raw Cap   ║ Usable Cap   ║ Efficiency║\n")
	fmt.Fprintf(w, " ╠═══════════╬═══════════╬═══════════╬═══════════╬═══════════╬══════════════╬═══════════╣\n")

	for _, r := range results {
		fmt.Fprintf(w, " ║ %-9s ║ %-9d ║ %-9d ║ %-9d ║ %-9s ║ %-12s ║ %6.1f%%   ║\n",
			r.Name,
			r.Result.Nodes,
			r.Result.OSDsPerNode,
			r.Result.TotalOSDs,
			calc.FormatBytes(r.Result.RawCapacity),
			calc.FormatBytes(r.Result.UsableCapacity),
			r.Result.Efficiency,
		)
	}

	fmt.Fprintf(w, " ╚═══════════╩═══════════╩═══════════╩═══════════╩═══════════╩══════════════╩═══════════╝\n")
	fmt.Fprintf(w, "\n")
}

func PrintCephadmSpec(result calc.Result, w io.Writer) {
	if w == nil {
		w = os.Stdout
	}
	fmt.Fprintf(w, "%s\n", GenerateCephadmSpec(result))
}
