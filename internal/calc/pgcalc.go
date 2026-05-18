package calc

import (
	"math"
)

func suggestPGs(totalOSDs int, input Input) []PGSuggestion {
	targetPGsPerOSD := 100

	var suggestions []PGSuggestion

	if input.Protection == ProtectionEC {
		suggestions = append(suggestions, PGSuggestion{
			PoolName: "ec-data",
			PGCount:  calcPGs(totalOSDs, 0.70, targetPGsPerOSD),
			IsEC:     true,
		})
		suggestions = append(suggestions, PGSuggestion{
			PoolName: "repl-metadata",
			PGCount:  calcPGs(totalOSDs, 0.10, targetPGsPerOSD),
		})
		suggestions = append(suggestions, PGSuggestion{
			PoolName: ".rgw.root",
			PGCount:  calcPGs(totalOSDs, 0.05, targetPGsPerOSD),
		})
		suggestions = append(suggestions, PGSuggestion{
			PoolName: "rgw.rgw.buckets.data",
			PGCount:  calcPGs(totalOSDs, 0.10, targetPGsPerOSD),
		})
		suggestions = append(suggestions, PGSuggestion{
			PoolName: "rgw.rgw.buckets.index",
			PGCount:  calcPGs(totalOSDs, 0.05, targetPGsPerOSD),
		})
	} else {
		suggestions = append(suggestions, PGSuggestion{
			PoolName: "rbd",
			PGCount:  calcPGs(totalOSDs, 0.50, targetPGsPerOSD),
		})
		suggestions = append(suggestions, PGSuggestion{
			PoolName: "cephfs-metadata",
			PGCount:  calcPGs(totalOSDs, 0.10, targetPGsPerOSD),
		})
		suggestions = append(suggestions, PGSuggestion{
			PoolName: "cephfs-data",
			PGCount:  calcPGs(totalOSDs, 0.30, targetPGsPerOSD),
		})
		suggestions = append(suggestions, PGSuggestion{
			PoolName: ".rgw.root",
			PGCount:  calcPGs(totalOSDs, 0.05, targetPGsPerOSD),
		})
		suggestions = append(suggestions, PGSuggestion{
			PoolName: "rgw.rgw.buckets.data",
			PGCount:  calcPGs(totalOSDs, 0.05, targetPGsPerOSD),
		})
	}

	return suggestions
}

func calcPGs(totalOSDs int, dataPercent float64, targetPGsPerOSD int) int {
	raw := float64(targetPGsPerOSD) * float64(totalOSDs) * dataPercent
	if raw < float64(totalOSDs) {
		raw = float64(totalOSDs)
	}
	pg := nearestPowerOf2(int(math.Ceil(raw)))
	if pg < 32 {
		pg = 32
	}
	return pg
}

func nearestPowerOf2(n int) int {
	if n <= 1 {
		return 1
	}
	p := 1
	for p < n {
		p *= 2
	}
	lower := p / 2
	if float64(n-lower)/float64(lower) > 0.25 {
		return p
	}
	return lower
}
