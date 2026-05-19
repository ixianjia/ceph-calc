# Lipgloss TUI Refactoring Design

## Goal

Refactor the interactive CLI wizard in `internal/tui/tui.go` to use `charmbracelet/lipgloss` for styling, replacing raw ANSI escape codes and plain text formatting with structured styles.

## Scope

- **No logic changes** — all prompt/scan/calculate flow stays identical
- **No bubbletea migration** — keep `bufio.Scanner` sequential wizard
- **New file:** `internal/tui/style.go` — central style definitions
- **Modified file:** `internal/tui/tui.go` — use style constants from `style.go`

It is a mixed refactoring: keep the logical structure but significantly beautify the visual presentation.

## Style System (`style.go`)

A single file that defines all reusable styles and a theme color palette.

### Color Palette

| Token | Light Term | Dark Term | Usage |
|-------|-----------|-----------|-------|
| `accent` | #005FCF | #58A6FF | Step titles, section headers |
| `secondary` | #656D76 | #8B949E | Dim/hint text, footnotes |
| `success` | #1A7F37 | #3FB950 | Results, positive values |
| `warning` | #9A6700 | #D29922 | Warnings |
| `border` | #D0D7DE | #30363D | Box borders |
| `text` | #1F2328 | #E6EDF3 | Primary text |

Uses `lipgloss.AdaptiveColor` for automatic light/dark terminal adaptation.

### Defined Styles

- **HeaderStyle** — banner box with border (rounded/ double), centered, accent foreground
- **StepStyle** — bold + accent foreground, e.g. `Step 1/9: Target Usable Capacity`
- **InputLabelStyle** — right-aligned label column for prompt lines
- **DimStyle** — secondary foreground, italic (replaces `\033[2m` raw ANSI)
- **BoldStyle** — bold weight (replaces `\033[1m` raw ANSI)
- **MenuOptionStyle** — indented menu items (e.g. `1) Replication x3`)
- **SelectedStyle** — accent foreground for selected values
- **ResultSectionStyle** — section title in result output, bold + accent
- **ResultLabelStyle** — key column left-aligned (e.g. "Nodes:")
- **ResultValueStyle** — value column right-aligned or emphasized
- **Separator** — full-width line via `lipgloss.NewStyle().Width(60).Border(lipgloss.NormalBorder())`

### Color map helpers

- `ReleaseColor(r CephRelease) lipgloss.Color` — subtle color per release
- `ProtectionLabel(p ProtectionMode, ...)` — formatted label string

## UI Changes (`tui.go`)

### Banner

Before:
```
  +-------------------------------------------+
  |         Ceph Capacity Planner             |
  +-------------------------------------------+
```

After: Rendered via `HeaderStyle.Render()` — lipgloss rounded border, centered text, accent color.

### Step Titles

Before:
```
  Step 1/9: Target Usable Capacity
```

After: `StepStyle.Render("Step 1/9: Target Usable Capacity")` — bold + accent.

### Prompts

Before:
```
  Capacity (e.g. 100TB, 1PB)    : 100TB
```

After: Label styled with `InputLabelStyle`, default value styled with `SelectedStyle`.

### Menu Options

Before:
```
  1)  HDD   (bulk storage, lowest cost)
  2)  SSD   (balanced performance)
  3)  NVMe  (high performance, DB/AI)
```

After: Each option bullet rendered with `MenuOptionStyle`. Number highlighted with accent.

### Result Output

Before: Plain `fmt.Sprintf` with `\033[1m` for bold section headers.

After:
- Section titles → `ResultSectionStyle.Render()`
- Key/value lines → aligned using `ResultLabelStyle` + `ResultValueStyle`
- Each section separated by `Separator`
- Efficiency % colored: green if ≥40%, yellow if ≥25%, otherwise warning

### Separators

Before: `strings.Repeat("-", 48)`

After: lipgloss border-based separator line with `Separator` style.

## Error Handling

- Input validation errors rendered in red via a `ErrorStyle` (should be rare since scanner defaults are reasonable)
- Parse errors already handled inline; just style the error message

## Testing

- No new test infrastructure needed (project has `go test ./...` with no tests yet)
- Manual verification: `go build && go run .` in interactive mode

## Out of Scope

- bubbletea migration (separate future effort if desired)
- Component system in `components/` directory
- Changing any calculation or export logic
- Adding new features to the wizard
