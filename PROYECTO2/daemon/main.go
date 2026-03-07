package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const procFile = "/proc/continfo_pr2_so1_202307705"

type ProcessInfo struct {
	PID     int
	PPID    int
	Name    string
	VSZKB   uint64
	RSSKB   uint64
	MemPct  uint64
	CPUTime uint64
}

type SystemInfo struct {
	RAMTotalKB uint64
	RAMFreeKB  uint64
	RAMUsedKB  uint64
	Processes  []ProcessInfo
}

func parseLine(line string, info *SystemInfo) error {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	switch {
	case strings.HasPrefix(line, "RAM_TOTAL_KB:"):
		value := strings.TrimPrefix(line, "RAM_TOTAL_KB:")
		n, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing RAM_TOTAL_KB: %w", err)
		}
		info.RAMTotalKB = n

	case strings.HasPrefix(line, "RAM_FREE_KB:"):
		value := strings.TrimPrefix(line, "RAM_FREE_KB:")
		n, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing RAM_FREE_KB: %w", err)
		}
		info.RAMFreeKB = n

	case strings.HasPrefix(line, "RAM_USED_KB:"):
		value := strings.TrimPrefix(line, "RAM_USED_KB:")
		n, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing RAM_USED_KB: %w", err)
		}
		info.RAMUsedKB = n

	case strings.HasPrefix(line, "PROC:"):
		value := strings.TrimPrefix(line, "PROC:")
		parts := strings.Split(value, "|")
		if len(parts) != 7 {
			return fmt.Errorf("invalid PROC line: %s", line)
		}

		pid, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("error parsing PID: %w", err)
		}

		ppid, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("error parsing PPID: %w", err)
		}

		vsz, err := strconv.ParseUint(parts[3], 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing VSZ: %w", err)
		}

		rss, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing RSS: %w", err)
		}

		memPct, err := strconv.ParseUint(parts[5], 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing MemPct: %w", err)
		}

		cpuTime, err := strconv.ParseUint(parts[6], 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing CPUTime: %w", err)
		}

		info.Processes = append(info.Processes, ProcessInfo{
			PID:     pid,
			PPID:    ppid,
			Name:    parts[2],
			VSZKB:   vsz,
			RSSKB:   rss,
			MemPct:  memPct,
			CPUTime: cpuTime,
		})
	}

	return nil
}

func readProcFile(path string) (*SystemInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open proc file: %w", err)
	}
	defer file.Close()

	info := &SystemInfo{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if err := parseLine(line, info); err != nil {
			return nil, err
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading proc file: %w", err)
	}

	return info, nil
}

type CPUResult struct {
	PID      int
	Name     string
	DeltaCPU uint64
	RSSKB    uint64
	VSZKB    uint64
	MemPct   uint64
}

func topByRSS(processes []ProcessInfo, n int) []ProcessInfo {
	copied := make([]ProcessInfo, len(processes))
	copy(copied, processes)

	sort.Slice(copied, func(i, j int) bool {
		return copied[i].RSSKB > copied[j].RSSKB
	})

	if len(copied) < n {
		n = len(copied)
	}
	return copied[:n]
}

func topCPUBetweenSamples(oldInfo, newInfo *SystemInfo, n int) []CPUResult {
	oldMap := make(map[int]ProcessInfo)
	for _, p := range oldInfo.Processes {
		oldMap[p.PID] = p
	}

	var results []CPUResult

	for _, newProc := range newInfo.Processes {
		oldProc, exists := oldMap[newProc.PID]
		if !exists {
			continue
		}

		if newProc.CPUTime >= oldProc.CPUTime {
			results = append(results, CPUResult{
				PID:      newProc.PID,
				Name:     newProc.Name,
				DeltaCPU: newProc.CPUTime - oldProc.CPUTime,
				RSSKB:    newProc.RSSKB,
				VSZKB:    newProc.VSZKB,
				MemPct:   newProc.MemPct,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].DeltaCPU > results[j].DeltaCPU
	})

	if len(results) < n {
		n = len(results)
	}
	return results[:n]
}

func main() {
	fmt.Println("Leyendo primera muestra...")
	first, err := readProcFile(procFile)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Println("Esperando 5 segundos para segunda muestra...")
	time.Sleep(5 * time.Second)

	fmt.Println("Leyendo segunda muestra...")
	second, err := readProcFile(procFile)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("=== RESUMEN DEL SISTEMA ===")
	fmt.Printf("RAM Total: %d KB\n", second.RAMTotalKB)
	fmt.Printf("RAM Free : %d KB\n", second.RAMFreeKB)
	fmt.Printf("RAM Used : %d KB\n", second.RAMUsedKB)
	fmt.Printf("Procesos : %d\n\n", len(second.Processes))

	fmt.Println("=== TOP 5 POR RSS ===")
	for _, p := range topByRSS(second.Processes, 5) {
		fmt.Printf("PID=%d NAME=%s RSS=%d KB VSZ=%d KB MEM=%d CPU_TIME=%d\n",
			p.PID, p.Name, p.RSSKB, p.VSZKB, p.MemPct, p.CPUTime)
	}

	fmt.Println()
	fmt.Println("=== TOP 5 POR DELTA CPU ENTRE MUESTRAS ===")
	for _, p := range topCPUBetweenSamples(first, second, 5) {
		fmt.Printf("PID=%d NAME=%s DELTA_CPU=%d RSS=%d KB VSZ=%d KB MEM=%d\n",
			p.PID, p.Name, p.DeltaCPU, p.RSSKB, p.VSZKB, p.MemPct)
	}
}