package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ixianjia/ceph-calc/internal/calc"
)

var (
	accent    = lipgloss.AdaptiveColor{Light: "#005FCF", Dark: "#58A6FF"}
	secondary = lipgloss.AdaptiveColor{Light: "#656D76", Dark: "#8B949E"}
	green     = lipgloss.AdaptiveColor{Light: "#1A7F37", Dark: "#3FB950"}
	red       = lipgloss.AdaptiveColor{Light: "#CF222E", Dark: "#FF7B72"}
)

var (
	DimStyle    = lipgloss.NewStyle().Foreground(secondary)
	BoldStyle   = lipgloss.NewStyle().Bold(true)
	AccentStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	GreenStyle  = lipgloss.NewStyle().Foreground(green).Bold(true)
	RedStyle    = lipgloss.NewStyle().Foreground(red).Bold(true)

	Separator = BoldStyle.Foreground(secondary).Render(strings.Repeat("═", 52))
)

func ReleaseLabel(r calc.CephRelease) string {
	names := map[calc.CephRelease]string{
		calc.ReleasePacific: "Pacific (v16)",
		calc.ReleaseQuincy:  "Quincy (v17)",
		calc.ReleaseReef:    "Reef (v18)",
		calc.ReleaseSquid:   "Squid (v19)",
	}
	return names[r]
}

func ProtectionLabel(p calc.ProtectionMode, repl int, ecK, ecM int) string {
	if p == calc.ProtectionRepl {
		return fmt.Sprintf("Replication x%d", repl)
	}
	return fmt.Sprintf("EC %d+%d", ecK, ecM)
}

func DiskTypeLabel(d calc.DiskType) string {
	switch d {
	case calc.DiskHDD:
		return "HDD"
	case calc.DiskSSD:
		return "SSD"
	case calc.DiskNVME:
		return "NVMe"
	}
	return string(d)
}
