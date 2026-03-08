package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"os/exec"
	"bytes"

	"github.com/redis/go-redis/v9"
)

const procFile = "/proc/continfo_pr2_so1_202307705"
const logFile = "monitor_logs.jsonl"

var ctx = context.Background()

type ProcessInfo struct {
	PID     int    `json:"pid"`
	PPID    int    `json:"ppid"`
	Name    string `json:"name"`
	VSZKB   uint64 `json:"vsz_kb"`
	RSSKB   uint64 `json:"rss_kb"`
	MemPct  uint64 `json:"mem_pct"`
	CPUTime uint64 `json:"cpu_time"`
}

type SystemInfo struct {
	RAMTotalKB uint64        `json:"ram_total_kb"`
	RAMFreeKB  uint64        `json:"ram_free_kb"`
	RAMUsedKB  uint64        `json:"ram_used_kb"`
	Processes  []ProcessInfo `json:"processes"`
}

type CPUResult struct {
	PID      int    `json:"pid"`
	Name     string `json:"name"`
	DeltaCPU uint64 `json:"delta_cpu"`
	RSSKB    uint64 `json:"rss_kb"`
	VSZKB    uint64 `json:"vsz_kb"`
	MemPct   uint64 `json:"mem_pct"`
}

type CycleLog struct {
	Timestamp string        `json:"timestamp"`
	RAMTotal  uint64        `json:"ram_total_kb"`
	RAMFree   uint64        `json:"ram_free_kb"`
	RAMUsed   uint64        `json:"ram_used_kb"`
	TopRSS    []ProcessInfo `json:"top_rss"`
	TopCPU    []CPUResult   `json:"top_cpu_delta"`
}

type DockerContainer struct {
	ID    string
	Name  string
	Image string
	PID   int
}

type ContainerCandidate struct {
	Name     string
	Image    string
	PID      int
	DeltaCPU uint64
	RSSKB    uint64
	VSZKB    uint64
	MemPct   uint64
}

func getRunningContainers() ([]DockerContainer, error) {
	cmd := exec.Command("docker", "ps", "--format", "{{.ID}}|{{.Names}}|{{.Image}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := bytes.Split(bytes.TrimSpace(out), []byte("\n"))
	var containers []DockerContainer

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		parts := strings.Split(string(line), "|")
		if len(parts) != 3 {
			continue
		}

		id := parts[0]
		name := parts[1]
		image := parts[2]

		inspectCmd := exec.Command("docker", "inspect", "-f", "{{.State.Pid}}", id)
		pidOut, err := inspectCmd.Output()
		if err != nil {
			continue
		}

		pidStr := strings.TrimSpace(string(pidOut))
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		containers = append(containers, DockerContainer{
			ID:    id,
			Name:  name,
			Image: image,
			PID:   pid,
		})
	}

	return containers, nil
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

func candidateProcessesByCPU(oldInfo, newInfo *SystemInfo, n int) []CPUResult {
	results := topCPUBetweenSamples(oldInfo, newInfo, len(newInfo.Processes))
	var filtered []CPUResult

	protected := map[string]bool{
		"grafana":     true,
		"valkey":      true,
		"systemd":     true,
		"gnome-shell": true,
	}

	for _, p := range results {
		if protected[p.Name] {
			continue
		}
		filtered = append(filtered, p)
		if len(filtered) == n {
			break
		}
	}

	return filtered
}

func candidateContainersByCPU(oldInfo, newInfo *SystemInfo, containers []DockerContainer, n int) []ContainerCandidate {
	processCandidates := topCPUBetweenSamples(oldInfo, newInfo, len(newInfo.Processes))

	containerMap := make(map[int]DockerContainer)
	for _, c := range containers {
		containerMap[c.PID] = c
	}

	protectedContainers := map[string]bool{
		"grafana_so1": true,
		"valkey_so1":  true,
	}

	var results []ContainerCandidate

	for _, p := range processCandidates {
		c, exists := containerMap[p.PID]
		if !exists {
			continue
		}

		if protectedContainers[c.Name] {
			continue
		}

		if strings.HasPrefix(c.Name, "low_") {
			continue
		}

		results = append(results, ContainerCandidate{
			Name:     c.Name,
			Image:    c.Image,
			PID:      c.PID,
			DeltaCPU: p.DeltaCPU,
			RSSKB:    p.RSSKB,
			VSZKB:    p.VSZKB,
			MemPct:   p.MemPct,
		})

		if len(results) == n {
			break
		}
	}

	return results
}

func removeContainer(name string) error {
	cmd := exec.Command("docker", "rm", "-f", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error removing container %s: %v - %s", name, err, string(output))
	}
	return nil
}

func appendJSONLog(path string, entry CycleLog) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	return enc.Encode(entry)
}

func saveToValkey(rdb *redis.Client, entry CycleLog) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	if err := rdb.RPush(ctx, "monitor:logs", data).Err(); err != nil {
		return err
	}

	if err := rdb.Set(ctx, "monitor:ram_total", entry.RAMTotal, 0).Err(); err != nil {
		return err
	}

	if err := rdb.Set(ctx, "monitor:ram_free", entry.RAMFree, 0).Err(); err != nil {
		return err
	}

	if err := rdb.Set(ctx, "monitor:ram_used", entry.RAMUsed, 0).Err(); err != nil {
		return err
	}

	if err := rdb.Set(ctx, "monitor:last_update", entry.Timestamp, 0).Err(); err != nil {
		return err
	}

	topRSSJSON, err := json.Marshal(entry.TopRSS)
	if err != nil {
		return err
	}
	if err := rdb.Set(ctx, "monitor:top_rss", topRSSJSON, 0).Err(); err != nil {
		return err
	}

	topCPUJSON, err := json.Marshal(entry.TopCPU)
	if err != nil {
		return err
	}
	if err := rdb.Set(ctx, "monitor:top_cpu", topCPUJSON, 0).Err(); err != nil {
		return err
	}

	return nil
}

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Println("Error connecting to Valkey:", err)
		os.Exit(1)
	}

	fmt.Println("Conectado a Valkey correctamente")

	for {
		fmt.Println("======================================")
		fmt.Println("Leyendo primera muestra...")
		first, err := readProcFile(procFile)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		fmt.Println("Esperando 10 segundos para segunda muestra...")
		time.Sleep(10 * time.Second)

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

		containers, err := getRunningContainers()
		if err != nil {
			fmt.Println("Error obteniendo contenedores Docker:", err)
		} else {
			fmt.Println("=== CONTENEDORES ACTIVOS ===")
			for _, c := range containers {
				fmt.Printf("NAME=%s IMAGE=%s PID=%d\n", c.Name, c.Image, c.PID)
			}
			fmt.Println()
		}

		topRSS := topByRSS(second.Processes, 5)
		topCPU := topCPUBetweenSamples(first, second, 5)
		candidates := candidateProcessesByCPU(first, second, 5)

		var containerCandidates []ContainerCandidate
		if err == nil {
			containerCandidates = candidateContainersByCPU(first, second, containers, 5)
		}

		fmt.Println("=== TOP 5 POR RSS ===")
		for _, p := range topRSS {
			fmt.Printf("PID=%d NAME=%s RSS=%d KB VSZ=%d KB MEM=%d CPU_TIME=%d\n",
				p.PID, p.Name, p.RSSKB, p.VSZKB, p.MemPct, p.CPUTime)
		}

		fmt.Println()
		fmt.Println("=== TOP 5 POR DELTA CPU ENTRE MUESTRAS ===")
		for _, p := range topCPU {
			fmt.Printf("PID=%d NAME=%s DELTA_CPU=%d RSS=%d KB VSZ=%d KB MEM=%d\n",
				p.PID, p.Name, p.DeltaCPU, p.RSSKB, p.VSZKB, p.MemPct)
		}

		fmt.Println()
		fmt.Println("=== POSIBLES CANDIDATOS A ELIMINAR ===")
		for _, p := range candidates {
			fmt.Printf("PID=%d NAME=%s DELTA_CPU=%d RSS=%d KB VSZ=%d KB MEM=%d\n",
				p.PID, p.Name, p.DeltaCPU, p.RSSKB, p.VSZKB, p.MemPct)
		}

		fmt.Println()
		fmt.Println("=== CONTENEDORES CANDIDATOS A ELIMINAR ===")
		for _, c := range containerCandidates {
			fmt.Printf("NAME=%s IMAGE=%s PID=%d DELTA_CPU=%d RSS=%d KB VSZ=%d KB MEM=%d\n",
				c.Name, c.Image, c.PID, c.DeltaCPU, c.RSSKB, c.VSZKB, c.MemPct)
		}

		if len(containerCandidates) > 0 {
			target := containerCandidates[0]
			fmt.Println()
			fmt.Printf("Eliminando contenedor candidato: %s\n", target.Name)

			if err := removeContainer(target.Name); err != nil {
				fmt.Println("Error al eliminar contenedor:", err)
			} else {
				fmt.Printf("Contenedor eliminado correctamente: %s\n", target.Name)
			}
		} else {
			fmt.Println()
			fmt.Println("No hay contenedores candidatos para eliminar.")
		}

		entry := CycleLog{
			Timestamp: time.Now().Format(time.RFC3339),
			RAMTotal:  second.RAMTotalKB,
			RAMFree:   second.RAMFreeKB,
			RAMUsed:   second.RAMUsedKB,
			TopRSS:    topRSS,
			TopCPU:    topCPU,
		}

		if err := appendJSONLog(logFile, entry); err != nil {
			fmt.Println("Error writing JSON log:", err)
		} else {
			fmt.Printf("\nLog guardado en %s\n", logFile)
		}

		if err := saveToValkey(rdb, entry); err != nil {
			fmt.Println("Error saving to Valkey:", err)
		} else {
			fmt.Println("Log guardado en Valkey (lista monitor:logs)")
		}

		fmt.Println()
		fmt.Println("Esperando 10 segundos para el siguiente ciclo...")
		time.Sleep(10 * time.Second)
	}
}