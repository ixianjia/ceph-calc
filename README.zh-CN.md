# ceph-calc

Ceph 容量规划 CLI 工具 — 输入目标可用容量，输出完整的硬件方案和生产级 cephadm spec。

![演示](demo.gif)

```bash
# 交互式向导（9 步，零依赖）
ceph-calc

# 一键计算
ceph-calc --usable 100TB --repl 3

# 对比所有副本策略
ceph-calc --usable 100TB --compare

# 生成完整 cephadm spec 和部署指南
ceph-calc --usable 100TB --repl 3 --services 3 --gen-spec > cluster.yaml
```

[English 🇬🇧](README.md)

## 功能

- **反向容量计算**: 输入 `100TB 可用` → 输出 `3 节点 × 8 × 16TB HDD = 384TB 裸容量`
- **冗余策略**: 副本 x3 / 纠删码（K+M 可配置）
- **硬件估算（依据 Ceph 官方文档）**:
  - CPU: HDD 每 OSD 3 线程，NVMe 每 OSD 6 线程
  - 内存: `(OSDs × osd_memory_target × 2 + OS) × 1.2` 余量
  - DB/WAL 分离推荐（HDD 场景）
  - 网络速率/网段根据 OSD 密度和盘型推荐
- **附加服务**: CephFS（MDS）、RGW，或两者同时开启 — 自动调整 CPU/内存估算
- **Ceph 版本感知**: Pacific/Quincy/Reef/Squid — 版本相关的 PG 配置建议
- **完整 cephadm spec**: config、hosts、mon、mgr、osd、pools、mds、rgw、crash、grafana、prometheus
- **部署指南**: spec 文件中嵌入逐步部署说明
- **策略对比**: 6 种策略横向对比
- **预设场景**: `small`、`medium`、`large`、`all-nvme`
- **导出格式**: 表格（默认）、JSON、YAML
- **纯文本交互**: 使用 `fmt` + `bufio.Scanner`，任何终端均可运行

## 安装

```bash
go install github.com/ixianjia/ceph-calc@latest

# 或从源码构建
git clone https://github.com/ixianjia/ceph-calc.git
cd ceph-calc
go build -o ceph-calc .
sudo mv ceph-calc /usr/local/bin/
```

依赖: Go 1.22+

## 用法

### 交互式向导

```bash
ceph-calc
```

按提示逐步输入，直接回车使用默认值：

```
  Step 1/9: Target Usable Capacity
  Capacity (e.g. 100TB, 1PB) : 100TB

  Step 2/9: Protection Strategy
    1)  Replication x3  (3 copies, 66% overhead)
    2)  Erasure Coding  (EC K+M, configurable)
  Choose [1-2] : 1
  ...
```

### 非交互模式

```bash
# 基础用法
ceph-calc --usable 100TB --repl 3

# 纠删码
ceph-calc --usable 100TB --ec-k 4 --ec-m 2

# 带 CephFS+RGW、Reef 版本、8×NVMe
ceph-calc --usable 100TB --repl 3 --services 3 --release 3 --disk 4TB --disk-type nvme

# JSON 输出
ceph-calc --usable 100TB --repl 3 -o json
```

### 生成 cephadm spec

```bash
ceph-calc --usable 100TB --repl 3 --services 3 --gen-spec > cluster.yaml

# spec 文件包含完整部署指南：
head -30 cluster.yaml
```

部署:
```bash
cephadm bootstrap --mon-ip 10.0.1.10
ceph orch apply -i cluster.yaml
```

### 策略对比

```bash
ceph-calc --usable 100TB --compare
```

输出:
```
 ╔═══════════╦═══════════╦═══════════╦══════════════════════════════╗
 ║ 策略      ║ 节点      ║ 裸容量    ║ 可用容量       ║ 效率      ║
 ╠═══════════╬═══════════╬═══════════╬════════════════╬═══════════╣
 ║ Repl x2   ║ 3         ║ 288.0 TB  ║ 115.2 TB       ║   40.0%   ║
 ║ Repl x3   ║ 3         ║ 384.0 TB  ║ 102.4 TB       ║   26.7%   ║
 ║ EC 2+1    ║ 3         ║ 192.0 TB  ║ 102.4 TB       ║   53.3%   ║
 ║ EC 4+2    ║ 3         ║ 192.0 TB  ║ 102.4 TB       ║   53.3%   ║
 ║ EC 6+2    ║ 3         ║ 192.0 TB  ║ 115.2 TB       ║   60.0%   ║
 ║ EC 8+3    ║ 3         ║ 192.0 TB  ║ 111.7 TB       ║   58.2%   ║
 ╚═══════════╩═══════════╩═══════════╩════════════════╩═══════════╝
```

### 预设场景

```bash
ceph-calc --preset small       # 50TB, 3副本, 8TB HDD
ceph-calc --preset medium      # 200TB, 3副本, 16TB HDD
ceph-calc --preset large       # 1PB, EC 4+2, 16TB HDD
ceph-calc --preset all-nvme    # 100TB, 3副本, 4TB NVMe
```

## 参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-u, --usable` | — | 目标可用容量（如 `100TB`、`1PB`） |
| `-r, --repl` | `3` | 副本数 |
| `--ec-k` | — | 纠删码数据块数 |
| `--ec-m` | — | 纠删码校验块数 |
| `--disk` | `16TB` | 单盘容量 |
| `--disk-type` | `hdd` | 盘类型: `hdd`、`ssd`、`nvme` |
| `--disks-per-node` | `12` | 每节点最大盘数 |
| `--safety` | `0.80` | 安全水位 (0.50-0.95) |
| `-n, --nodes` | `0` | 固定节点数（`0` = 自动） |
| `--services` | `0` | 附加服务: `0`=仅OSD, `1`=+CephFS, `2`=+RGW, `3`=+CephFS+RGW |
| `--release` | `2` (Reef) | Ceph 版本: `0`=Pacific, `1`=Quincy, `2`=Reef, `3`=Squid |
| `--compare` | `false` | 对比所有保护策略 |
| `--gen-spec` | `false` | 生成 cephadm spec YAML |
| `--preset` | — | 预设场景: `small`、`medium`、`large`、`all-nvme` |
| `-o, --output` | `table` | 输出格式: `table`、`json`、`yaml` |

## 计算原理

### 容量公式

```
可用容量 = 裸容量 / 冗余系数 × 安全水位

副本 x3:  系数 = 3
EC 4+2:   系数 = (4+2)/4 = 1.5
EC 6+2:   系数 = (6+2)/6 = 1.33
```

### 硬件估算（依据 Ceph 官方文档）

| 组件 | HDD | SSD | NVMe |
|------|-----|-----|------|
| CPU 线程/OSD | 3（最少 1，推荐 3） | 4 | 6（最少 4，推荐 6） |
| 内存公式 | OSD×4GB×2+系统, ×1.2 | 同左 | OSD×6GB×2.5+系统, ×1.2 |
| DB/WAL 分离 | 每 HDD OSD 配 SSD | 不需要 | 不需要 |
| 集群网络 | 10-25GbE | 25GbE | 25-100GbE |

内存公式来自官方: `总内存 > (OSDs × osd_memory_target(4GB) × 2)`，另加 20% recovery 余量。

### PG 计算

- 目标: 每 OSD 约 100 个 PG
- 各池: `PG数 = 目标 × 总OSD数 × 数据比例`
- 结果: 取最近 2 的幂（最小 32）
- 建议: 启用 `pg_autoscale_mode on`（Reef 起默认开启）

## 许可

MIT
