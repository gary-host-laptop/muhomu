package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"muhomu/widgets"
)

// ── system stats ──────────────────────────────────────────────
var (
	prevNetRx, prevNetTx uint64
	prevNetTime          time.Time
	prevIdle, prevTotal  uint64
)

type statsPoint struct {
	CPU float64 `json:"cpu"`
	RAM float64 `json:"ram"`
}

var (
	statsMu      sync.RWMutex
	statsCurrent struct {
		cpu, ram, rx, tx float64
	}
	statsHistory []statsPoint
	// statsMax is the ring size: 30 visible slots + 1 extra point that the
	// system-stats widget renders one slot off-screen right. At each swap
	// the fresh line is then pixel-identical to the previous frame's
	// resting position, so the CSS scroll glide can slide the newest point
	// in from the right while the oldest slides off-screen left — no point
	// is ever dropped at a swap (see statsGraph in widgets/system_stats.go).
	statsMax = 31
)

// startStatsCollector launches a background goroutine that polls /proc
// files every 2 seconds and keeps a ring buffer of history.
func startStatsCollector() {
	go func() {
		for {
			cpu := getCPUPercent()
			ram := getRAMPercent()
			rx, tx := getNetSpeeds()

			statsMu.Lock()
			statsCurrent.cpu = cpu
			statsCurrent.ram = ram
			statsCurrent.rx = rx
			statsCurrent.tx = tx
			statsHistory = append(statsHistory, statsPoint{CPU: cpu, RAM: ram})
			if len(statsHistory) > statsMax {
				statsHistory = statsHistory[len(statsHistory)-statsMax:]
			}
			statsMu.Unlock()

			time.Sleep(2 * time.Second)
		}
	}()
}

// ── rss feed warmer ───────────────────────────────────────────
// startRSSWarmer periodically refreshes every configured rss widget's
// feeds (per-feed TTL / ETag rules apply as usual), so articles stay
// fresh even while no browser is open and full-page renders — which
// never touch the network — almost always find warm cache entries.
// The config walk re-reads the current config on every tick, so
// hot-reloaded feed lists are picked up without a restart.
func startRSSWarmer() {
	warmAll := func() {
		for _, wcfg := range flattenWidgetConfigs(allWidgetConfigs()) {
			if wcfg.ID != "rss" {
				continue
			}
			widgets.WarmRSSCache(wcfg.Options, appCfg.faviconDir)
		}
	}

	// Warm immediately at startup rather than after one period.
	warmAll()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		warmAll()
	}
}

func getCPUPercent() float64 {
	f, _ := os.Open("/proc/stat")
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	parts := strings.Fields(scanner.Text())
	if len(parts) < 8 {
		return 0
	}
	user, _ := strconv.ParseUint(parts[1], 10, 64)
	nice, _ := strconv.ParseUint(parts[2], 10, 64)
	system, _ := strconv.ParseUint(parts[3], 10, 64)
	idle, _ := strconv.ParseUint(parts[4], 10, 64)
	iowait, _ := strconv.ParseUint(parts[5], 10, 64)
	irq, _ := strconv.ParseUint(parts[6], 10, 64)
	softirq, _ := strconv.ParseUint(parts[7], 10, 64)

	total := user + nice + system + idle + iowait + irq + softirq
	if prevTotal == 0 {
		prevIdle = idle
		prevTotal = total
		return 0
	}
	deltaTotal := float64(total - prevTotal)
	deltaIdle := float64(idle - prevIdle)
	cpu := 100 * (deltaTotal - deltaIdle) / deltaTotal
	prevIdle = idle
	prevTotal = total
	return cpu
}

func getRAMPercent() float64 {
	f, _ := os.Open("/proc/meminfo")
	defer f.Close()
	scanner := bufio.NewScanner(f)
	total, available := uint64(0), uint64(0)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			total, _ = strconv.ParseUint(strings.Fields(line)[1], 10, 64)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			available, _ = strconv.ParseUint(strings.Fields(line)[1], 10, 64)
		}
	}
	if total == 0 {
		return 0
	}
	return 100 * float64(total-available) / float64(total)
}

func getNetSpeeds() (rx, tx float64) {
	f, _ := os.Open("/proc/net/dev")
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var rxBytes, txBytes uint64
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, ":") && !strings.Contains(line, "lo") {
			parts := strings.Fields(line)
			if len(parts) >= 10 {
				rxBytes, _ = strconv.ParseUint(parts[1], 10, 64)
				txBytes, _ = strconv.ParseUint(parts[9], 10, 64)
				break
			}
		}
	}
	now := time.Now()
	if prevNetTime.IsZero() {
		prevNetRx = rxBytes
		prevNetTx = txBytes
		prevNetTime = now
		return 0, 0
	}
	dt := now.Sub(prevNetTime).Seconds()
	if dt == 0 {
		return 0, 0
	}
	rx = float64(rxBytes-prevNetRx) / dt / 1024
	tx = float64(txBytes-prevNetTx) / dt / 1024
	prevNetRx = rxBytes
	prevNetTx = txBytes
	prevNetTime = now
	return rx, tx
}

// currentSystemStats returns a consistent snapshot of the latest stats
// (values + history) under the stats lock, for the system-stats widget.
func currentSystemStats() widgets.SystemStats {
	statsMu.RLock()
	defer statsMu.RUnlock()
	hist := make([]widgets.StatsPoint, len(statsHistory))
	for i, p := range statsHistory {
		hist[i] = widgets.StatsPoint{CPU: p.CPU, RAM: p.RAM}
	}
	return widgets.SystemStats{
		CPU:     statsCurrent.cpu,
		RAM:     statsCurrent.ram,
		RX:      statsCurrent.rx,
		TX:      statsCurrent.tx,
		History: hist,
	}
}
