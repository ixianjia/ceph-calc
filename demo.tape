Output demo.gif
Set FontSize 16
Set Width 720
Set Height 432
Set FontFamily "JetBrains Mono"
Set Theme "Dracula"
Set Padding 24
Set Framerate 10
Set PlaybackSpeed 1.8

# Step 1: Show help
Type "/home/chirs/Projects/ceph-calc/ceph-calc --help" Sleep 300ms
Enter
Sleep 2s
Ctrl+l

# Step 2: Non-interactive quick calc
Type "/home/chirs/Projects/ceph-calc/ceph-calc --usable 100TB --repl 3" Sleep 200ms
Enter
Sleep 3s
Ctrl+l

# Step 3: Compare all strategies
Type "/home/chirs/Projects/ceph-calc/ceph-calc --usable 100TB --compare" Sleep 200ms
Enter
Sleep 3s
Ctrl+l

# Step 4: Generate cephadm spec
Type "/home/chirs/Projects/ceph-calc/ceph-calc --usable 100TB --repl 3 --services 3 --gen-spec | head -25" Sleep 200ms
Enter
Sleep 4s
Ctrl+l

# Step 5: Interactive demo - fast mode
Type "/home/chirs/Projects/ceph-calc/ceph-calc" Sleep 300ms
Enter
Sleep 800ms

# Step 5.1: Usable capacity
Type "100TB" Sleep 100ms
Enter
Sleep 300ms

# Step 5.2: Protection (Repl x3 - default)
Enter
Sleep 300ms

# Step 5.3: Disk type (HDD - default)
Enter
Sleep 300ms

# Step 5.4: Disk size (16TB - default)
Enter
Sleep 300ms

# Step 5.5: Disks per node (12 - default)
Enter
Sleep 300ms

# Step 5.6: Safety ratio (0.80 - default)
Enter
Sleep 300ms

# Step 5.7: Services (OSD only - default)
Enter
Sleep 300ms

# Step 5.8: Ceph release (Reef - default)
Enter
Sleep 300ms

# Step 5.9: Node count (auto - default)
Enter
Sleep 2s

# Let the result show
Sleep 5s
