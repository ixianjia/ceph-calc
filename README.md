# ceph-calc

Ceph capacity planning CLI tool — input target usable capacity, get a complete hardware plan and production-ready cephadm spec.

![demo](demo.gif)

```bash
# Interactive wizard (9 steps, zero dependencies)
ceph-calc

# One-shot calculation
ceph-calc --usable 100TB --repl 3

# Compare all strategies side by side
ceph-calc --usable 100TB --compare

# Generate full cephadm spec with deployment guide
ceph-calc --usable 100TB --repl 3 --services 3 --gen-spec > cluster.yaml
```

[中文文档 🇨🇳](README.zh-CN.md)

## Features

- **Reverse capacity calculation**: input `100TB usable` → output `3 nodes × 8 HDD 16TB = 384TB raw`
- **Protection strategies**: Replication x3 / Erasure Coding (K+M configurable)
- **Hardware estimation per Ceph official docs**:
  - CPU: 3 threads/HDD OSD, 6 threads/NVMe OSD (official recommendations)
  - RAM: `(OSDs × osd_memory_target × 2 + OS) × 1.2` headroom
  - DB/WAL offload recommendation for HDD
  - Network speed/CIDR based on OSD density + disk type
- **Additional services**: CephFS (MDS), RGW, or both — adjusts CPU/RAM estimate
- **Ceph release awareness**: Pacific/Quincy/Reef/Squid — version-specific PG guidance
- **Full cephadm spec generation**: config, hosts, mon, mgr, osd, pools, mds, rgw, crash, grafana, prometheus
- **Deployment guide**: step-by-step instructions embedded in the spec file
- **Strategy comparison**: 6 strategies side by side
- **Presets**: `small`, `medium`, `large`, `all-nvme`
- **Export formats**: table (default), JSON, YAML
- **Zero TUI framework**: uses `fmt` + `bufio.Scanner` — works in any terminal

## Installation

```bash
go install github.com/ixianjia/ceph-calc@latest

# Or build from source
git clone https://github.com/ixianjia/ceph-calc.git
cd ceph-calc
go build -o ceph-calc .
sudo mv ceph-calc /usr/local/bin/
```

Prerequisites: Go 1.22+

## Usage

### Interactive wizard

```bash
ceph-calc
```

Follow the 9-step prompts. Press `Enter` to accept defaults:

```
  Step 1/9: Target Usable Capacity
  Capacity (e.g. 100TB, 1PB) : 100TB

  Step 2/9: Protection Strategy
    1)  Replication x3  (3 copies, 66% overhead)
    2)  Erasure Coding  (EC K+M, configurable)
  Choose [1-2] : 1
  ...
```

### Non-interactive mode

```bash
# Basic
ceph-calc --usable 100TB --repl 3

# Erasure coding
ceph-calc --usable 100TB --ec-k 4 --ec-m 2

# With CephFS + RGW, Reef release, 8xNVMe
ceph-calc --usable 100TB --repl 3 --services 3 --release 3 --disk 4TB --disk-type nvme

# JSON output
ceph-calc --usable 100TB --repl 3 -o json
```

### Generate cephadm spec

```bash
ceph-calc --usable 100TB --repl 3 --services 3 --gen-spec > cluster.yaml

# The spec includes a complete deployment guide:
head -30 cluster.yaml
```

Deploy with:
```bash
cephadm bootstrap --mon-ip 10.0.1.10
ceph orch apply -i cluster.yaml
```

### Compare strategies

```bash
ceph-calc --usable 100TB --compare
```

Output:
```
 ╔═══════════╦═══════════╦═══════════╦══════════════════════════════╗
 ║ Strategy  ║  Nodes    ║ Raw Cap   ║ Usable Cap   ║ Efficiency  ║
 ╠═══════════╬═══════════╬═══════════╬══════════════╬══════════════╣
 ║ Repl x2   ║ 3         ║ 288.0 TB  ║ 115.2 TB     ║   40.0%     ║
 ║ Repl x3   ║ 3         ║ 384.0 TB  ║ 102.4 TB     ║   26.7%     ║
 ║ EC 2+1    ║ 3         ║ 192.0 TB  ║ 102.4 TB     ║   53.3%     ║
 ║ EC 4+2    ║ 3         ║ 192.0 TB  ║ 102.4 TB     ║   53.3%     ║
 ║ EC 6+2    ║ 3         ║ 192.0 TB  ║ 115.2 TB     ║   60.0%     ║
 ║ EC 8+3    ║ 3         ║ 192.0 TB  ║ 111.7 TB     ║   58.2%     ║
 ╚═══════════╩═══════════╩═══════════╩══════════════╩══════════════╝
```

### Presets

```bash
ceph-calc --preset small       # 50TB, 3x repl, 8TB HDD
ceph-calc --preset medium      # 200TB, 3x repl, 16TB HDD
ceph-calc --preset large       # 1PB, EC 4+2, 16TB HDD
ceph-calc --preset all-nvme    # 100TB, 3x repl, 4TB NVMe
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-u, --usable` | — | Target usable capacity (e.g. `100TB`, `1PB`) |
| `-r, --repl` | `3` | Replication factor |
| `--ec-k` | — | Erasure coding data chunks |
| `--ec-m` | — | Erasure coding parity chunks |
| `--disk` | `16TB` | Disk size |
| `--disk-type` | `hdd` | Disk type: `hdd`, `ssd`, `nvme` |
| `--disks-per-node` | `12` | Max disks per node |
| `--safety` | `0.80` | Safety ratio (0.50-0.95) |
| `-n, --nodes` | `0` | Fixed node count (`0` = auto) |
| `--services` | `0` | Additional services: `0`=OSD only, `1`=+CephFS, `2`=+RGW, `3`=+CephFS+RGW |
| `--release` | `2` (Reef) | Ceph release: `0`=Pacific, `1`=Quincy, `2`=Reef, `3`=Squid |
| `--compare` | `false` | Compare all protection strategies |
| `--gen-spec` | `false` | Generate cephadm spec YAML |
| `--preset` | — | Use preset: `small`, `medium`, `large`, `all-nvme` |
| `-o, --output` | `table` | Output format: `table`, `json`, `yaml` |

## How it works

### Capacity calculation

```
usable = raw / protection_overhead × safety_ratio

Replica x3:  overhead = 3
EC 4+2:      overhead = (4+2)/4 = 1.5
EC 6+2:      overhead = (6+2)/6 = 1.33
```

### Hardware estimation (per Ceph official docs)

| Component | HDD | SSD | NVMe |
|-----------|-----|-----|------|
| CPU threads/OSD | 3 (min 1, rec 3) | 4 | 6 (min 4, rec 6) |
| RAM factor | OSD×4GB×2+OS, ×1.2 | same | OSD×6GB×2.5+OS, ×1.2 |
| DB/WAL offload | SSD per HDD OSD | not needed | not needed |
| Cluster network | 10-25GbE | 25GbE | 25-100GbE |

RAM formula from official docs: `total > (OSDs × osd_memory_target(4GB) × 2)`, plus 20% extra for recovery spikes.

### PG calculation

- Target: ~100 PGs per OSD
- Each pool: `PGs = target × total OSDs × data_share`
- Result: nearest power of 2 (minimum 32)
- Recommendation: enable `pg_autoscale_mode on` (default since Reef)

## License

MIT
