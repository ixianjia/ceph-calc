# Lipgloss TUI Refactoring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) for syntax tracking.

**Goal:** Replace raw ANSI escape codes and plain formatting in `internal/tui/tui.go` with charmbracelet/lipgloss styles, keeping all wizard logic unchanged.

**Architecture:** Add a `style.go` defining a theme + reusable styles (header, steps, dim, bold, result sections, separators) with automatic light/dark terminal adaptation. Refactor `tui.go` to consume these styles.

**Tech Stack:** Go 1.24, charmbracelet/lipgloss v1.1.0, no new dependencies.

---

### Task 1: Create style.go with theme and style definitions

**Files:**
- Create: `internal/tui/style.go`
- No test file (purely visual constants)

- [ ] **Step 1: Write style.go**

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ixianjia/ceph-calc/internal/calc"
)

// Color palette — adaptive to terminal light/dark
var (
	accent    = lipgloss.AdaptiveColor{Light: "#005FCF", Dark: "#58A6FF"}
	secondary = lipgloss.AdaptiveColor{Light: "#656D76", Dark: "#8B949E"}
	success   = lipgloss.AdaptiveColor{Light: "#1A7F37", Dark: "#3FB950"}
	warning   = lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#D29922"}
	border    = lipgloss.AdaptiveColor{Light: "#D0D7DE", Dark: "#30363D"}
	text      = lipgloss.AdaptiveColor{Light: "#1F2328", Dark: "#E6EDF3"}
	subtleBG  = lipgloss.AdaptiveColor{Light: "#F6F8FA", Dark: "#161B22"}
)

// Reusable styles
var (
	HeaderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 2).
			Align(lipgloss.Center).
			Bold(true).
			Foreground(accent).
			Width(45)

	StepStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent)

	InputLabelStyle = lipgloss.NewStyle().
			Width(38).
			Align(lipgloss.Left)

	DimStyle = lipgloss.NewStyle().
			Foreground(secondary).
			Italic(true)

	BoldStyle = lipgloss.NewStyle().Bold(true)

	MenuOptionStyle = lipgloss.NewStyle().
			Foreground(text)

	SelectedStyle = lipgloss.NewStyle().
			Foreground(accent).
			Bold(true)

	ResultSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accent).
				Underline(true)

	ResultLabelStyle = lipgloss.NewStyle().
			Width(20).
			Align(lipgloss.Right).
			Foreground(secondary)

	ResultValueStyle = lipgloss.NewStyle().
			Foreground(text)

	ResultValueAccent = lipgloss.NewStyle().
				Foreground(success).
				Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#CF222E", Dark: "#FF7B72"}).
			Bold(true)

	Separator = lipgloss.NewStyle().
			Foreground(border).
			Width(52).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(border).
			Render("")
)

func NumberedOption(n int, label string) string {
	return fmt.Sprintf("%s  %s", DimStyle.Render(fmt.Sprintf("%d)", n)), label)
}

func ReleaseLabel(r calc.CephRelease) string {
	names := map[calc.CephRelease]string{
		calc.ReleasePacific: "Pacific (v16)",
		calc.ReleaseQuincy:  "Quincy (v17)",
		calc.ReleaseReef:    "Reef (v18)",
		calc.ReleaseSquid:   "Squid (v19)",
	}
	return names[r]
}

func ServicesLabel(s calc.AdditionalService) string {
	labels := map[calc.AdditionalService]string{
		calc.ServiceOSDOnly: "OSD only (baseline)",
		calc.ServiceCephFS:  "+ CephFS (MDS)",
		calc.ServiceRGW:     "+ RGW (object gateway)",
		calc.ServiceBoth:    "+ CephFS + RGW",
	}
	return labels[s]
}

func ProtectionLabel(p calc.ProtectionMode, repl int, ecK, ecM int) string {
	if p == calc.ProtectionRepl {
		return fmt.Sprintf("Replication x%d", repl)
	}
	return fmt.Sprintf("EC %d+%d", ecK, ecM)
}

func EfficiencyColor(eff float64) lipgloss.Style {
	switch {
	case eff >= 40:
		return ResultValueAccent
	case eff >= 25:
		return lipgloss.NewStyle().Foreground(warning).Bold(true)
	default:
		return ErrorStyle
	}
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
```

- [ ] **Step 2: Verify compilation**

Run: `go vet ./internal/tui/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/tui/style.go
git commit -m "feat: add lipgloss theme and style definitions"
```

---

### Task 2: Refactor tui.go to use lipgloss styles

**Files:**
- Modify: `internal/tui/tui.go` (replace `dim()`, `bold()`, all `fmt.Printf` formatting, result rendering)

- [ ] **Step 1: Rewrite tui.go**

Replace the entire content of `internal/tui/tui.go` with the styled version:

```go
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
	fmt.Printf("  %s : %s ", InputLabelStyle.Render(strings.TrimSpace(label)), SelectedStyle.Render(val))
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
	fmt.Println("  " + HeaderStyle.Render("Ceph Capacity Planner"))
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
	fmt.Println("  " + StepStyle.Render("Step 1/9: Target Usable Capacity"))
	in.usable = prompt(scanner, "Capacity (e.g. 100TB, 1PB)", "100TB")
	if _, err := calc.ParseCapacity(in.usable); err != nil {
		fmt.Printf("\n  %s\n", ErrorStyle.Render("Invalid: "+err.Error()))
		return
	}
	fmt.Println()

	// Step 2: Protection strategy
	fmt.Println("  " + StepStyle.Render("Step 2/9: Protection Strategy"))
	fmt.Printf("    %s\n", NumberedOption(1, "Replication x3  (3 copies, 66% overhead)"))
	fmt.Printf("    %s\n", NumberedOption(2, "Erasure Coding  (EC K+M, configurable)"))
	protStr := prompt(scanner, "Choose [1-2]", "1")
	switch strings.TrimSpace(protStr) {
	case "2":
		in.protection = calc.ProtectionEC
		fmt.Println()
		fmt.Println("  " + DimStyle.Render("Step 2b: EC Parameters (K/M)"))
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
	fmt.Println("  " + StepStyle.Render("Step 3/9: Disk Type"))
	fmt.Printf("    %s\n", NumberedOption(1, "HDD   (bulk storage, lowest cost)"))
	fmt.Printf("    %s\n", NumberedOption(2, "SSD   (balanced performance)"))
	fmt.Printf("    %s\n", NumberedOption(3, "NVMe  (high performance, DB/AI)"))
	dtStr := prompt(scanner, "Choose [1-3]", "1")
	switch strings.TrimSpace(dtStr) {
	case "2":
		in.diskType = calc.DiskSSD
	case "3":
		in.diskType = calc.DiskNVME
	}
	fmt.Println()

	// Step 4: Disk size
	fmt.Println("  " + StepStyle.Render("Step 4/9: Disk Size"))
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
		fmt.Printf("%s  %s  ", DimStyle.Render(fmt.Sprintf("%2d)", i+1)), s.label)
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
	fmt.Println("  " + StepStyle.Render("Step 5/9: Disks per Node"))
	dpn := prompt(scanner, "Max disks per node [1-36]", "12")
	in.disksPerNode, _ = strconv.Atoi(dpn)
	if in.disksPerNode < 1 || in.disksPerNode > 36 {
		in.disksPerNode = 12
	}
	fmt.Println()

	// Step 6: Safety ratio
	fmt.Println("  " + StepStyle.Render("Step 6/9: Safety Ratio"))
	sr := prompt(scanner, "Safety ratio [0.50-0.95, default 0.80]", "0.80")
	in.safetyRatio, _ = strconv.ParseFloat(sr, 64)
	if in.safetyRatio < 0.50 || in.safetyRatio > 0.95 {
		in.safetyRatio = 0.80
	}
	fmt.Println()

	// Step 7: Additional services
	fmt.Println("  " + StepStyle.Render("Step 7/9: Additional Services"))
	fmt.Printf("    %s\n", NumberedOption(1, "OSD only (baseline)"))
	fmt.Printf("    %s\n", NumberedOption(2, "+ CephFS  (MDS, file system metadata)"))
	fmt.Printf("    %s\n", NumberedOption(3, "+ RGW     (object gateway)"))
	fmt.Printf("    %s\n", NumberedOption(4, "+ CephFS + RGW"))
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
	fmt.Println("  " + StepStyle.Render("Step 8/9: Ceph Release"))
	fmt.Printf("    %s\n", NumberedOption(1, "Pacific (v16) — minor"))
	fmt.Printf("    %s\n", NumberedOption(2, "Quincy (v17) — stable"))
	fmt.Printf("    %s\n", NumberedOption(3, "Reef (v18, recommended)"))
	fmt.Printf("    %s\n", NumberedOption(4, "Squid (v19) — latest, Crimson OSD"))
	rlsStr := prompt(scanner, "Choose [1-4]", "3")
	switch strings.TrimSpace(rlsStr) {
	case "1":
		in.release = calc.ReleasePacific
	case "2":
		in.release = calc.ReleaseQuincy
	case "3":
		in.release = calc.ReleaseReef
	case "4":
		in.release = calc.ReleaseSquid
	}
	fmt.Println()

	// Step 9: Node count
	fmt.Println("  " + StepStyle.Render("Step 9/9: Node Count"))
	nc := prompt(scanner, "Node count (0=auto)", "0")
	in.nodeCount, _ = strconv.Atoi(nc)
	if in.nodeCount < 0 {
		in.nodeCount = 0
	}
	fmt.Println()

	// Calculate after all inputs
	fmt.Println("  " + DimStyle.Render("Calculating..."))
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

	fmt.Println(renderResult(in.usable, calcResult, sizes[szIdx-1].label))

	fmt.Println()
	fmt.Println("  " + DimStyle.Render("ctrl+c to exit"))
	fmt.Scanln()
}

func renderResult(usable string, r calc.Result, diskLabel string) string {
	prot := ProtectionLabel(r.Input.Protection, r.Input.ReplFactor, r.Input.ECK, r.Input.ECM)

	var b strings.Builder

	b.WriteString("\n  " + Separator + "\n")

	// Target section
	b.WriteString("  " + ResultSectionStyle.Render("Target") + "\n")
	b.WriteString(fmt.Sprintf("  %s %s\n", ResultLabelStyle.Render("Usable:"), ResultValueStyle.Render(usable)))
	b.WriteString(fmt.Sprintf("  %s %s\n", ResultLabelStyle.Render("Protection:"), ResultValueStyle.Render(prot)))
	b.WriteString(fmt.Sprintf("  %s %s %s\n", ResultLabelStyle.Render("Disk:"), ResultValueStyle.Render(DiskTypeLabel(r.Input.DiskType)), ResultValueStyle.Render(calc.FormatBytes(r.Input.DiskSize))))
	b.WriteString(fmt.Sprintf("  %s %s\n", ResultLabelStyle.Render("Safety:"), ResultValueStyle.Render(fmt.Sprintf("%.0f%%", r.Input.SafetyRatio*100))))
	b.WriteString(fmt.Sprintf("  %s %s\n", ResultLabelStyle.Render("Release:"), ResultValueStyle.Render(ReleaseLabel(r.Input.CephRelease))))

	b.WriteString("\n  " + Separator + "\n")

	// Hardware plan section
	b.WriteString("  " + ResultSectionStyle.Render("Hardware Plan") + "\n")
	b.WriteString(fmt.Sprintf("  %s %s\n", ResultLabelStyle.Render("Nodes:"), ResultValueAccent.Render(fmt.Sprintf("%d", r.Nodes))))
	b.WriteString(fmt.Sprintf("  %s %s\n", ResultLabelStyle.Render("OSDs per node:"), ResultValueStyle.Render(fmt.Sprintf("%d", r.OSDsPerNode))))
	b.WriteString(fmt.Sprintf("  %s %s\n", ResultLabelStyle.Render("Total OSDs:"), ResultValueStyle.Render(fmt.Sprintf("%d", r.TotalOSDs))))
	b.WriteString(fmt.Sprintf("  %s %s\n", ResultLabelStyle.Render("Raw:"), ResultValueStyle.Render(calc.FormatBytes(r.RawCapacity))))
	b.WriteString(fmt.Sprintf("  %s %s\n", ResultLabelStyle.Render("Usable:"), ResultValueStyle.Render(calc.FormatBytes(r.UsableCapacity))))
	b.WriteString(fmt.Sprintf("  %s %s\n", ResultLabelStyle.Render("Efficiency:"), EfficiencyColor(r.Efficiency).Render(fmt.Sprintf("%.1f%%", r.Efficiency))))

	b.WriteString("\n  " + Separator + "\n")

	// Node spec section
	b.WriteString("  " + ResultSectionStyle.Render("Node Spec (per node)") + "\n")
	b.WriteString(fmt.Sprintf("  %s %s\n", ResultLabelStyle.Render("CPU:"), ResultValueStyle.Render(fmt.Sprintf("%d cores (%s)", r.Hardware.CPUCoresPerNode, r.Hardware.CPUDesc))))
	b.WriteString(fmt.Sprintf("  %s %s\n", ResultLabelStyle.Render("RAM:"), ResultValueStyle.Render(fmt.Sprintf("%.0f GB (%s)", r.Hardware.RAMGBPerNode, r.Hardware.RAMDesc))))
	b.WriteString(fmt.Sprintf("  %s %s\n", ResultLabelStyle.Render("OSDs:"), ResultValueStyle.Render(fmt.Sprintf("%d x %s", r.OSDsPerNode, calc.FormatBytes(r.Input.DiskSize)))))
	b.WriteString(fmt.Sprintf("  %s %s\n", ResultLabelStyle.Render("Network:"), ResultValueStyle.Render(fmt.Sprintf("%s public + %s cluster", r.Network.PublicSpeed, r.Network.ClusterSpeed))))
	b.WriteString(fmt.Sprintf("  %s %s\n", ResultLabelStyle.Render("Services:"), ResultValueStyle.Render(r.Hardware.ServicesDesc)))

	b.WriteString("\n  " + Separator + "\n")

	// DB/WAL
	b.WriteString("  " + ResultSectionStyle.Render("DB/WAL") + "\n")
	b.WriteString("  " + DimStyle.Render("  "+r.Hardware.DBWALDesc) + "\n")

	b.WriteString("\n  " + Separator + "\n")

	// Network plan
	b.WriteString("  " + ResultSectionStyle.Render("Network Plan") + "\n")
	b.WriteString(fmt.Sprintf("  %s %s  %s\n", ResultLabelStyle.Render("Public:"), ResultValueStyle.Render(r.Network.PublicSpeed), DimStyle.Render(r.Network.PublicNetwork)))
	b.WriteString(fmt.Sprintf("  %s %s  %s\n", ResultLabelStyle.Render("Cluster:"), ResultValueStyle.Render(r.Network.ClusterSpeed), DimStyle.Render(r.Network.ClusterNetwork)))

	b.WriteString("\n  " + Separator + "\n")

	// PG suggestion
	b.WriteString("  " + ResultSectionStyle.Render("PG Suggestion") + "\n")
	for _, pg := range r.PGSuggestions {
		b.WriteString(fmt.Sprintf("    %-26s %s\n", pg.PoolName, ResultValueAccent.Render(fmt.Sprintf("%d PGs", pg.PGCount))))
	}
	if r.Hardware.PGConfigDesc != "" {
		for _, line := range strings.Split(r.Hardware.PGConfigDesc, "\n") {
			b.WriteString("  " + DimStyle.Render("  "+line) + "\n")
		}
	}

	b.WriteString("\n  " + Separator + "\n")

	// Reusable CLI command
	b.WriteString("  " + ResultSectionStyle.Render("Reusable CLI Command") + "\n")
	cmd := "ceph-calc"
	cmd += " --usable " + usable
	if r.Input.Protection == calc.ProtectionRepl {
		cmd += fmt.Sprintf(" --repl %d", r.Input.ReplFactor)
	} else {
		cmd += fmt.Sprintf(" --ec-k %d --ec-m %d", r.Input.ECK, r.Input.ECM)
	}
	cmd += fmt.Sprintf(" --disk %s --disk-type %s", strings.TrimSpace(diskLabel), r.Input.DiskType)
	cmd += fmt.Sprintf(" --disks-per-node %d", r.Input.DisksPerNode)
	b.WriteString(fmt.Sprintf("  %s\n", DimStyle.Render("  "+cmd)))

	b.WriteString("\n  " + Separator + "\n")

	// Generate cephadm spec hint
	b.WriteString("  " + ResultSectionStyle.Render("Generate Cephadm Spec") + "\n")
	b.WriteString(fmt.Sprintf("  %s\n", DimStyle.Render("  "+cmd+" --gen-spec > cluster.yaml")))

	b.WriteString("\n  " + Separator + "\n")

	return b.String()
}
```

- [ ] **Step 2: Verify compilation**

Run: `go vet ./internal/tui/`
Expected: no errors

- [ ] **Step 3: Remove unused dim/bold functions**

The old `dim()` and `bold()` functions are no longer referenced. Verify by building the whole project.

Run: `go build -o /dev/null .`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add internal/tui/tui.go
git commit -m "refactor: use lipgloss styles for all TUI rendering"
```

---

### Task 3: Build and visual verification

- [ ] **Step 1: Full build and vet**

Run:
```bash
go vet ./...
go build -o /tmp/ceph-calc .
```
Expected: clean exit, binary at `/tmp/ceph-calc`

- [ ] **Step 2: Verify non-interactive mode still works**

Run: `/tmp/ceph-calc --usable 100TB --repl 3`
Expected: correct output, no raw ANSI codes leaking in non-interactive output (verify the output doesn't use dim/bold from tui since it uses export paths)

- [ ] **Step 3: Final commit**

```bash
git add -A
git commit -m "chore: build and verify lipgloss TUI refactor" || true
```
